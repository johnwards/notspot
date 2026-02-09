package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/domain"
)

// SearchStore defines the interface for CRM search operations.
type SearchStore interface {
	Search(ctx context.Context, objectType string, req *domain.SearchRequest) (*domain.SearchResult, error)
}

// SQLiteSearchStore implements SearchStore backed by SQLite.
type SQLiteSearchStore struct {
	db *sql.DB
}

// NewSQLiteSearchStore creates a new SQLiteSearchStore.
func NewSQLiteSearchStore(db *sql.DB) *SQLiteSearchStore {
	return &SQLiteSearchStore{db: db}
}

const (
	maxFilterGroups    = 5
	maxFiltersPerGroup = 6
	maxSearchLimit     = 200
	maxSearchTotal     = 10000
)

// defaultSearchableProps are properties searched by the "query" field.
var defaultSearchableProps = []string{
	"email", "firstname", "lastname", "name", "domain", "company",
	"hs_object_id", "phone", "website",
}

// Search executes a CRM search with filters, sorts, and pagination.
func (s *SQLiteSearchStore) Search(ctx context.Context, objectType string, req *domain.SearchRequest) (*domain.SearchResult, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), ErrNotFound)
	}

	if err := validateSearchRequest(req); err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	offset := 0
	if req.After != "" {
		parsed, err := strconv.Atoi(req.After)
		if err != nil {
			return nil, &ValidationError{Message: "invalid after cursor"}
		}
		offset = parsed
	}

	if offset+limit > maxSearchTotal {
		limit = maxSearchTotal - offset
		if limit <= 0 {
			return &domain.SearchResult{
				Total:   0,
				Results: []*domain.Object{},
			}, nil
		}
	}

	// Build the shared FROM + WHERE clause used by both count and select.
	fromClause, whereClause, baseArgs, sortAlias, err := buildSearchClauses(typeID, req)
	if err != nil {
		return nil, err
	}

	// Count query.
	countSQL := "SELECT COUNT(DISTINCT o.id)" + fromClause + whereClause
	var total int
	if err := s.db.QueryRowContext(ctx, countSQL, baseArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("search count: %w", err)
	}
	if total > maxSearchTotal {
		total = maxSearchTotal
	}

	// Select query with ORDER BY + LIMIT/OFFSET.
	selectSQL := "SELECT DISTINCT o.id" + fromClause + whereClause
	selectArgs := make([]any, len(baseArgs))
	copy(selectArgs, baseArgs)

	if sortAlias != "" && len(req.Sorts) > 0 {
		direction := "ASC"
		if strings.EqualFold(req.Sorts[0].Direction, "DESCENDING") {
			direction = "DESC"
		}
		selectSQL += fmt.Sprintf(" ORDER BY %s.value %s, o.id ASC", sortAlias, direction)
	} else {
		selectSQL += " ORDER BY o.id ASC"
	}
	selectSQL += " LIMIT ? OFFSET ?"
	selectArgs = append(selectArgs, limit, offset)

	rows, err := s.db.QueryContext(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var objectIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		objectIDs = append(objectIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}

	// Fetch full objects with properties.
	results := make([]*domain.Object, 0, len(objectIDs))
	for _, id := range objectIDs {
		var obj domain.Object
		var archivedAt sql.NullString
		err := s.db.QueryRowContext(ctx,
			`SELECT id, archived, archived_at, created_at, updated_at FROM objects WHERE id = ?`, id,
		).Scan(&obj.ID, &obj.Archived, &archivedAt, &obj.CreatedAt, &obj.UpdatedAt)
		if err != nil {
			continue
		}
		if archivedAt.Valid {
			obj.ArchivedAt = archivedAt.String
		}

		obj.Properties, err = s.getProperties(ctx, id, req.Properties)
		if err != nil {
			return nil, err
		}

		results = append(results, &obj)
	}

	result := &domain.SearchResult{
		Total:   total,
		Results: results,
	}

	nextOffset := offset + limit
	if nextOffset < total {
		result.Paging = &domain.SearchPaging{
			Next: domain.SearchPagingNext{
				After: strconv.Itoa(nextOffset),
			},
		}
	}

	return result, nil
}

// getProperties fetches property values for an object.
func (s *SQLiteSearchStore) getProperties(ctx context.Context, objectID string, props []string) (map[string]string, error) {
	var rows *sql.Rows
	var err error

	if len(props) == 0 {
		rows, err = s.db.QueryContext(ctx,
			`SELECT property_name, value FROM property_values WHERE object_id = ?`,
			objectID,
		)
	} else {
		allProps := make(map[string]bool)
		for _, p := range defaultProps {
			allProps[p] = true
		}
		for _, p := range props {
			allProps[p] = true
		}

		placeholders := make([]string, 0, len(allProps))
		args := make([]any, 0, len(allProps)+1)
		args = append(args, objectID)
		for p := range allProps {
			placeholders = append(placeholders, "?")
			args = append(args, p)
		}
		rows, err = s.db.QueryContext(ctx,
			`SELECT property_name, value FROM property_values WHERE object_id = ? AND property_name IN (`+strings.Join(placeholders, ",")+`)`,
			args...,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("get properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		result[name] = value
	}
	return result, rows.Err()
}

// ValidationError represents a search validation error.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func validateSearchRequest(req *domain.SearchRequest) error {
	if req.Limit > maxSearchLimit {
		return &ValidationError{
			Message: fmt.Sprintf("limit must be less than or equal to %d", maxSearchLimit),
		}
	}
	if len(req.FilterGroups) > maxFilterGroups {
		return &ValidationError{
			Message: fmt.Sprintf("maximum of %d filter groups allowed", maxFilterGroups),
		}
	}
	for i, group := range req.FilterGroups {
		if len(group.Filters) > maxFiltersPerGroup {
			return &ValidationError{
				Message: fmt.Sprintf("maximum of %d filters per group allowed (group %d)", maxFiltersPerGroup, i),
			}
		}
		for _, f := range group.Filters {
			if f.PropertyName == "" {
				return &ValidationError{Message: "filter propertyName is required"}
			}
			if !isValidOperator(f.Operator) {
				return &ValidationError{
					Message: fmt.Sprintf("invalid operator: %s", f.Operator),
				}
			}
		}
	}
	return nil
}

func isValidOperator(op string) bool {
	switch op {
	case "EQ", "NEQ", "LT", "LTE", "GT", "GTE",
		"BETWEEN", "IN", "NOT_IN",
		"HAS_PROPERTY", "NOT_HAS_PROPERTY",
		"CONTAINS_TOKEN", "NOT_CONTAINS_TOKEN":
		return true
	}
	return false
}

// buildSearchClauses builds the FROM and WHERE portions of the search query,
// returning them along with the ordered args and the sort join alias (if any).
func buildSearchClauses(typeID string, req *domain.SearchRequest) (fromClause, whereClause string, args []any, sortAlias string, err error) {
	var fromSB strings.Builder
	var whereSB strings.Builder
	filterIdx := 0

	fromSB.WriteString(" FROM objects o")

	// Add LEFT JOINs for each filter property.
	for _, group := range req.FilterGroups {
		for _, f := range group.Filters {
			alias := fmt.Sprintf("pv_f%d", filterIdx)
			fmt.Fprintf(&fromSB, " LEFT JOIN property_values %s ON %s.object_id = o.id AND %s.property_name = ?",
				alias, alias, alias)
			args = append(args, f.PropertyName)
			filterIdx++
		}
	}

	// Add LEFT JOIN for query (full-text search across searchable properties).
	queryAlias := ""
	if req.Query != "" {
		queryAlias = fmt.Sprintf("pv_q%d", filterIdx)
		fmt.Fprintf(&fromSB, " LEFT JOIN property_values %s ON %s.object_id = o.id", queryAlias, queryAlias)
		filterIdx++
	}

	// Add LEFT JOIN for sort property.
	if len(req.Sorts) > 0 {
		sortAlias = fmt.Sprintf("pv_s%d", filterIdx)
		fmt.Fprintf(&fromSB, " LEFT JOIN property_values %s ON %s.object_id = o.id AND %s.property_name = ?",
			sortAlias, sortAlias, sortAlias)
		args = append(args, req.Sorts[0].PropertyName)
	}

	// WHERE clause base.
	whereSB.WriteString(" WHERE o.object_type_id = ? AND o.archived = FALSE")
	args = append(args, typeID)

	// Add filter conditions.
	if len(req.FilterGroups) > 0 {
		filterIdx = 0
		var groupClauses []string
		for _, group := range req.FilterGroups {
			var filterClauses []string
			for i := range group.Filters {
				alias := fmt.Sprintf("pv_f%d", filterIdx)
				clause, filterArgs, buildErr := buildFilterClause(alias, &group.Filters[i])
				if buildErr != nil {
					err = buildErr
					return
				}
				filterClauses = append(filterClauses, clause)
				args = append(args, filterArgs...)
				filterIdx++
			}
			groupClauses = append(groupClauses, "("+strings.Join(filterClauses, " AND ")+")")
		}
		whereSB.WriteString(" AND (")
		whereSB.WriteString(strings.Join(groupClauses, " OR "))
		whereSB.WriteString(")")
	}

	// Add query condition.
	if req.Query != "" {
		propPlaceholders := make([]string, len(defaultSearchableProps))
		for i, p := range defaultSearchableProps {
			propPlaceholders[i] = "?"
			args = append(args, p)
		}
		fmt.Fprintf(&whereSB, " AND %s.property_name IN (%s)", queryAlias, strings.Join(propPlaceholders, ","))
		fmt.Fprintf(&whereSB, " AND %s.value LIKE ?", queryAlias)
		args = append(args, "%"+req.Query+"%")
	}

	fromClause = fromSB.String()
	whereClause = whereSB.String()
	return
}

func buildFilterClause(alias string, f *domain.Filter) (clause string, args []any, err error) {
	switch f.Operator {
	case "EQ":
		return fmt.Sprintf("%s.value = ?", alias), []any{f.Value}, nil
	case "NEQ":
		return fmt.Sprintf("(%s.value IS NULL OR %s.value != ?)", alias, alias), []any{f.Value}, nil
	case "LT":
		return fmt.Sprintf("%s.value < ?", alias), []any{f.Value}, nil
	case "LTE":
		return fmt.Sprintf("%s.value <= ?", alias), []any{f.Value}, nil
	case "GT":
		return fmt.Sprintf("%s.value > ?", alias), []any{f.Value}, nil
	case "GTE":
		return fmt.Sprintf("%s.value >= ?", alias), []any{f.Value}, nil
	case "BETWEEN":
		return fmt.Sprintf("%s.value BETWEEN ? AND ?", alias), []any{f.Value, f.HighValue}, nil
	case "IN":
		if len(f.Values) == 0 {
			return "1=0", nil, nil
		}
		placeholders := make([]string, len(f.Values))
		fArgs := make([]any, len(f.Values))
		for i, v := range f.Values {
			placeholders[i] = "?"
			fArgs[i] = v
		}
		return fmt.Sprintf("%s.value IN (%s)", alias, strings.Join(placeholders, ",")), fArgs, nil
	case "NOT_IN":
		if len(f.Values) == 0 {
			return "1=1", nil, nil
		}
		placeholders := make([]string, len(f.Values))
		fArgs := make([]any, len(f.Values))
		for i, v := range f.Values {
			placeholders[i] = "?"
			fArgs[i] = v
		}
		return fmt.Sprintf("(%s.value IS NULL OR %s.value NOT IN (%s))", alias, alias, strings.Join(placeholders, ",")), fArgs, nil
	case "HAS_PROPERTY":
		return fmt.Sprintf("%s.value IS NOT NULL", alias), nil, nil
	case "NOT_HAS_PROPERTY":
		return fmt.Sprintf("%s.value IS NULL", alias), nil, nil
	case "CONTAINS_TOKEN":
		return fmt.Sprintf("%s.value LIKE ?", alias), []any{"%" + f.Value + "%"}, nil
	case "NOT_CONTAINS_TOKEN":
		return fmt.Sprintf("(%s.value IS NULL OR %s.value NOT LIKE ?)", alias, alias), []any{"%" + f.Value + "%"}, nil
	default:
		return "", nil, &ValidationError{Message: fmt.Sprintf("unsupported operator: %s", f.Operator)}
	}
}

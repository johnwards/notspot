package store_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

var _ store.SearchStore = (*store.SQLiteSearchStore)(nil)

func setupSearchStore(t *testing.T) (*store.SQLiteSearchStore, *store.SQLiteObjectStore) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	return store.NewSQLiteSearchStore(db), store.NewSQLiteObjectStore(db)
}

func TestSearchBasic(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	// Create test contacts.
	for _, email := range []string{"alice@example.com", "bob@example.com", "charlie@example.com"} {
		_, err := os.Create(ctx, "contacts", map[string]string{"email": email})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	// Search with no filters returns all.
	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected total=3, got %d", result.Total)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
}

func TestSearchEQFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, err := os.Create(ctx, "contacts", map[string]string{"email": "find@example.com", "firstname": "Find"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = os.Create(ctx, "contacts", map[string]string{"email": "skip@example.com", "firstname": "Skip"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "EQ", Value: "find@example.com"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestSearchNEQFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "NEQ", Value: "a@example.com"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchINFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "c@example.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "IN", Values: []string{"a@example.com", "c@example.com"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestSearchNOT_INFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "c@example.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "NOT_IN", Values: []string{"a@example.com", "c@example.com"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchContainsToken(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "bob@other.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "CONTAINS_TOKEN", Value: "example"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchHasProperty(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "firstname", Operator: "HAS_PROPERTY"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchNotHasProperty(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "firstname", Operator: "NOT_HAS_PROPERTY"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchBetween(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "age": "20"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com", "age": "30"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "c@example.com", "age": "40"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "age", Operator: "BETWEEN", Value: "25", HighValue: "35"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchMultipleFilterGroupsOR(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "bob@example.com", "firstname": "Bob"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "charlie@example.com", "firstname": "Charlie"})

	// Two filter groups (OR): email = alice OR firstname = Bob
	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "EQ", Value: "alice@example.com"},
				},
			},
			{
				Filters: []domain.Filter{
					{PropertyName: "firstname", Operator: "EQ", Value: "Bob"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestSearchMultipleFiltersAND(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice2@example.com", "firstname": "NotAlice"})

	// Single filter group with two filters (AND): email contains "alice" AND firstname = Alice
	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "CONTAINS_TOKEN", Value: "alice"},
					{PropertyName: "firstname", Operator: "EQ", Value: "Alice"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchPagination(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	for i := range 5 {
		_, err := os.Create(ctx, "contacts", map[string]string{
			"email": fmt.Sprintf("user%d@example.com", i),
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// First page.
	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{Limit: 2})
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Paging == nil {
		t.Fatal("expected paging")
	}
	if result.Paging.Next.After != "2" {
		t.Errorf("expected after=2, got %s", result.Paging.Next.After)
	}

	// Second page.
	result2, err := ss.Search(ctx, "contacts", &domain.SearchRequest{Limit: 2, After: "2"})
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	if len(result2.Results) != 2 {
		t.Errorf("expected 2 results on page 2, got %d", len(result2.Results))
	}

	// Third page.
	result3, err := ss.Search(ctx, "contacts", &domain.SearchRequest{Limit: 2, After: "4"})
	if err != nil {
		t.Fatalf("search page 3: %v", err)
	}
	if len(result3.Results) != 1 {
		t.Errorf("expected 1 result on page 3, got %d", len(result3.Results))
	}
	if result3.Paging != nil {
		t.Error("expected no paging on last page")
	}
}

func TestSearchQuery(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "bob@other.com", "firstname": "Bob"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{Query: "alice"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchSort(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "charlie@example.com", "firstname": "Charlie"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "alice@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "bob@example.com", "firstname": "Bob"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		Sorts:      []domain.Sort{{PropertyName: "firstname", Direction: "ASCENDING"}},
		Properties: []string{"firstname"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}
	if result.Results[0].Properties["firstname"] != "Alice" {
		t.Errorf("expected first=Alice, got %s", result.Results[0].Properties["firstname"])
	}
	if result.Results[2].Properties["firstname"] != "Charlie" {
		t.Errorf("expected last=Charlie, got %s", result.Results[2].Properties["firstname"])
	}
}

func TestSearchSortDescending(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "firstname": "Alice"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com", "firstname": "Bob"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "c@example.com", "firstname": "Charlie"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		Sorts:      []domain.Sort{{PropertyName: "firstname", Direction: "DESCENDING"}},
		Properties: []string{"firstname"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Results[0].Properties["firstname"] != "Charlie" {
		t.Errorf("expected first=Charlie, got %s", result.Results[0].Properties["firstname"])
	}
	if result.Results[2].Properties["firstname"] != "Alice" {
		t.Errorf("expected last=Alice, got %s", result.Results[2].Properties["firstname"])
	}
}

func TestSearchWithProperties(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "test@example.com", "firstname": "Test"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		Properties: []string{"email", "firstname"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Properties["email"] != "test@example.com" {
		t.Errorf("expected email in properties")
	}
	if result.Results[0].Properties["firstname"] != "Test" {
		t.Errorf("expected firstname in properties")
	}
}

func TestSearchValidationTooManyFilterGroups(t *testing.T) {
	ss, _ := setupSearchStore(t)
	ctx := context.Background()

	groups := make([]domain.FilterGroup, 6)
	for i := range groups {
		groups[i] = domain.FilterGroup{
			Filters: []domain.Filter{
				{PropertyName: "email", Operator: "EQ", Value: "test"},
			},
		}
	}

	_, err := ss.Search(ctx, "contacts", &domain.SearchRequest{FilterGroups: groups})
	if err == nil {
		t.Fatal("expected validation error for too many filter groups")
	}
	var ve *store.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchValidationTooManyFilters(t *testing.T) {
	ss, _ := setupSearchStore(t)
	ctx := context.Background()

	filters := make([]domain.Filter, 7)
	for i := range filters {
		filters[i] = domain.Filter{PropertyName: "email", Operator: "EQ", Value: "test"}
	}

	_, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{{Filters: filters}},
	})
	if err == nil {
		t.Fatal("expected validation error for too many filters")
	}
}

func TestSearchInvalidOperator(t *testing.T) {
	ss, _ := setupSearchStore(t)
	ctx := context.Background()

	_, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "email", Operator: "INVALID", Value: "test"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected validation error for invalid operator")
	}
}

func TestSearchNonexistentType(t *testing.T) {
	ss, _ := setupSearchStore(t)
	ctx := context.Background()

	_, err := ss.Search(ctx, "nonexistent", &domain.SearchRequest{})
	if err == nil {
		t.Fatal("expected error for nonexistent type")
	}
}

func TestSearchArchivedObjectsExcluded(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	obj, _ := os.Create(ctx, "contacts", map[string]string{"email": "archived@example.com"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "active@example.com"})
	_ = os.Archive(ctx, "contacts", obj.ID)

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1 (archived excluded), got %d", result.Total)
	}
}

func TestSearchGTFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "score": "10"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com", "score": "20"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "c@example.com", "score": "30"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "score", Operator: "GT", Value: "15"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestSearchLTFilter(t *testing.T) {
	ss, os := setupSearchStore(t)
	ctx := context.Background()

	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "a@example.com", "score": "10"})
	_, _ = os.Create(ctx, "contacts", map[string]string{"email": "b@example.com", "score": "20"})

	result, err := ss.Search(ctx, "contacts", &domain.SearchRequest{
		FilterGroups: []domain.FilterGroup{
			{
				Filters: []domain.Filter{
					{PropertyName: "score", Operator: "LT", Value: "15"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

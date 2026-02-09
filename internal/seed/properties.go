package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/johnwards/hubspot/internal/domain"
)

// defaultObjectTypes defines the standard HubSpot object types with their IDs.
var defaultObjectTypes = []struct {
	ID            string
	Name          string
	LabelSingular string
	LabelPlural   string
	PrimaryProp   string
}{
	{"0-1", "contacts", "Contact", "Contacts", "email"},
	{"0-2", "companies", "Company", "Companies", "name"},
	{"0-3", "deals", "Deal", "Deals", "dealname"},
	{"0-5", "tickets", "Ticket", "Tickets", "subject"},
	{"0-27", "tasks", "Task", "Tasks", "hs_task_subject"},
	{"0-46", "notes", "Note", "Notes", "hs_note_body"},
	{"0-47", "meetings", "Meeting", "Meetings", "hs_meeting_title"},
	{"0-48", "calls", "Call", "Calls", "hs_call_body"},
	{"0-49", "emails", "Email", "Emails", "hs_email_subject"},
}

type propDef struct {
	Name           string
	Label          string
	Type           string
	FieldType      string
	GroupName      string
	HasUniqueValue bool
}

var commonProps = []propDef{
	{Name: "hs_object_id", Label: "Object ID", Type: "number", FieldType: "number"},
	{Name: "hs_createdate", Label: "Create date", Type: "datetime", FieldType: "date"},
	{Name: "hs_lastmodifieddate", Label: "Last modified date", Type: "datetime", FieldType: "date"},
}

var objectProps = map[string][]propDef{
	"0-1": {
		{Name: "email", Label: "Email", Type: "string", FieldType: "text", GroupName: "contactinformation", HasUniqueValue: true},
		{Name: "firstname", Label: "First Name", Type: "string", FieldType: "text", GroupName: "contactinformation"},
		{Name: "lastname", Label: "Last Name", Type: "string", FieldType: "text", GroupName: "contactinformation"},
		{Name: "phone", Label: "Phone Number", Type: "string", FieldType: "phonenumber", GroupName: "contactinformation"},
		{Name: "company", Label: "Company Name", Type: "string", FieldType: "text", GroupName: "contactinformation"},
		{Name: "lifecyclestage", Label: "Lifecycle Stage", Type: "enumeration", FieldType: "radio", GroupName: "contactinformation"},
		{Name: "hubspot_owner_id", Label: "Owner", Type: "string", FieldType: "text", GroupName: "contactinformation"},
	},
	"0-2": {
		{Name: "name", Label: "Name", Type: "string", FieldType: "text", GroupName: "companyinformation"},
		{Name: "domain", Label: "Company Domain Name", Type: "string", FieldType: "text", GroupName: "companyinformation", HasUniqueValue: true},
		{Name: "industry", Label: "Industry", Type: "enumeration", FieldType: "select", GroupName: "companyinformation"},
		{Name: "lifecyclestage", Label: "Lifecycle Stage", Type: "enumeration", FieldType: "radio", GroupName: "companyinformation"},
		{Name: "hubspot_owner_id", Label: "Owner", Type: "string", FieldType: "text", GroupName: "companyinformation"},
	},
	"0-3": {
		{Name: "dealname", Label: "Deal Name", Type: "string", FieldType: "text", GroupName: "dealinformation"},
		{Name: "dealstage", Label: "Deal Stage", Type: "enumeration", FieldType: "radio", GroupName: "dealinformation"},
		{Name: "pipeline", Label: "Pipeline", Type: "enumeration", FieldType: "radio", GroupName: "dealinformation"},
		{Name: "amount", Label: "Amount", Type: "number", FieldType: "number", GroupName: "dealinformation"},
		{Name: "closedate", Label: "Close Date", Type: "date", FieldType: "date", GroupName: "dealinformation"},
		{Name: "hubspot_owner_id", Label: "Owner", Type: "string", FieldType: "text", GroupName: "dealinformation"},
	},
	"0-5": {
		{Name: "subject", Label: "Ticket Name", Type: "string", FieldType: "text", GroupName: "ticketinformation"},
		{Name: "content", Label: "Ticket Description", Type: "string", FieldType: "textarea", GroupName: "ticketinformation"},
		{Name: "hs_pipeline", Label: "Pipeline", Type: "enumeration", FieldType: "radio", GroupName: "ticketinformation"},
		{Name: "hs_pipeline_stage", Label: "Ticket Status", Type: "enumeration", FieldType: "radio", GroupName: "ticketinformation"},
		{Name: "hs_ticket_priority", Label: "Priority", Type: "enumeration", FieldType: "select", GroupName: "ticketinformation"},
		{Name: "hubspot_owner_id", Label: "Owner", Type: "string", FieldType: "text", GroupName: "ticketinformation"},
	},
	"0-27": {
		{Name: "hs_task_subject", Label: "Task Title", Type: "string", FieldType: "text", GroupName: "engagement_info"},
		{Name: "hs_task_body", Label: "Task Notes", Type: "string", FieldType: "textarea", GroupName: "engagement_info"},
		{Name: "hs_task_status", Label: "Task Status", Type: "enumeration", FieldType: "select", GroupName: "engagement_info"},
	},
	"0-46": {
		{Name: "hs_note_body", Label: "Note Body", Type: "string", FieldType: "textarea", GroupName: "engagement_info"},
	},
	"0-47": {
		{Name: "hs_meeting_title", Label: "Meeting Name", Type: "string", FieldType: "text", GroupName: "engagement_info"},
		{Name: "hs_meeting_start_time", Label: "Start Time", Type: "datetime", FieldType: "date", GroupName: "engagement_info"},
		{Name: "hs_meeting_end_time", Label: "End Time", Type: "datetime", FieldType: "date", GroupName: "engagement_info"},
	},
	"0-48": {
		{Name: "hs_call_body", Label: "Call Notes", Type: "string", FieldType: "textarea", GroupName: "engagement_info"},
		{Name: "hs_call_direction", Label: "Call Direction", Type: "enumeration", FieldType: "select", GroupName: "engagement_info"},
		{Name: "hs_call_duration", Label: "Call Duration", Type: "number", FieldType: "number", GroupName: "engagement_info"},
	},
	"0-49": {
		{Name: "hs_email_subject", Label: "Email Subject", Type: "string", FieldType: "text", GroupName: "engagement_info"},
		{Name: "hs_email_text", Label: "Email Body", Type: "string", FieldType: "textarea", GroupName: "engagement_info"},
	},
}

var defaultGroups = map[string]struct {
	Name  string
	Label string
}{
	"0-1":  {Name: "contactinformation", Label: "Contact Information"},
	"0-2":  {Name: "companyinformation", Label: "Company Information"},
	"0-3":  {Name: "dealinformation", Label: "Deal Information"},
	"0-5":  {Name: "ticketinformation", Label: "Ticket Information"},
	"0-27": {Name: "engagement_info", Label: "Task Information"},
	"0-46": {Name: "engagement_info", Label: "Note Information"},
	"0-47": {Name: "engagement_info", Label: "Meeting Information"},
	"0-48": {Name: "engagement_info", Label: "Call Information"},
	"0-49": {Name: "engagement_info", Label: "Email Information"},
}

// Properties inserts default object types, property groups, and property
// definitions. It is idempotent — existing rows are skipped.
func Properties(ctx context.Context, db *sql.DB) error {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	for _, ot := range defaultObjectTypes {
		_, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO object_types (id, name, label_singular, label_plural, primary_display_property, is_custom, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, FALSE, ?, ?)`,
			ot.ID, ot.Name, ot.LabelSingular, ot.LabelPlural, ot.PrimaryProp, ts, ts,
		)
		if err != nil {
			return fmt.Errorf("seed object type %s: %w", ot.Name, err)
		}
	}

	for typeID, grp := range defaultGroups {
		_, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO property_groups (object_type_id, name, label, display_order, archived)
			 VALUES (?, ?, ?, 0, FALSE)`,
			typeID, grp.Name, grp.Label,
		)
		if err != nil {
			return fmt.Errorf("seed property group %s: %w", grp.Name, err)
		}
	}

	emptyOpts, _ := json.Marshal([]domain.Option{})
	optsStr := string(emptyOpts)

	for _, ot := range defaultObjectTypes {
		grp := defaultGroups[ot.ID]

		for _, p := range commonProps {
			groupName := grp.Name
			_, err := db.ExecContext(ctx,
				`INSERT OR IGNORE INTO property_definitions (
					object_type_id, name, label, type, field_type, group_name,
					description, display_order, has_unique_value, hidden, form_field,
					calculated, external_options, hubspot_defined, options,
					archived, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, '', 0, FALSE, FALSE, FALSE, FALSE, FALSE, TRUE, ?, FALSE, ?, ?)`,
				ot.ID, p.Name, p.Label, p.Type, p.FieldType, groupName, optsStr, ts, ts,
			)
			if err != nil {
				return fmt.Errorf("seed common property %s for %s: %w", p.Name, ot.Name, err)
			}
		}

		for _, p := range objectProps[ot.ID] {
			_, err := db.ExecContext(ctx,
				`INSERT OR IGNORE INTO property_definitions (
					object_type_id, name, label, type, field_type, group_name,
					description, display_order, has_unique_value, hidden, form_field,
					calculated, external_options, hubspot_defined, options,
					archived, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, '', 0, ?, FALSE, FALSE, FALSE, FALSE, TRUE, ?, FALSE, ?, ?)`,
				ot.ID, p.Name, p.Label, p.Type, p.FieldType, p.GroupName,
				p.HasUniqueValue, optsStr, ts, ts,
			)
			if err != nil {
				return fmt.Errorf("seed property %s for %s: %w", p.Name, ot.Name, err)
			}
		}
	}

	return nil
}

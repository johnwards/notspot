package seed

// ObjectTypeDef defines a standard HubSpot object type.
type ObjectTypeDef struct {
	ID       string
	Singular string
	Plural   string
}

// StandardTypes maps object type names to their definitions.
var StandardTypes = map[string]ObjectTypeDef{
	"contacts":             {ID: "0-1", Singular: "Contact", Plural: "Contacts"},
	"companies":            {ID: "0-2", Singular: "Company", Plural: "Companies"},
	"deals":                {ID: "0-3", Singular: "Deal", Plural: "Deals"},
	"tickets":              {ID: "0-5", Singular: "Ticket", Plural: "Tickets"},
	"products":             {ID: "0-7", Singular: "Product", Plural: "Products"},
	"line_items":           {ID: "0-8", Singular: "Line Item", Plural: "Line Items"},
	"quotes":               {ID: "0-14", Singular: "Quote", Plural: "Quotes"},
	"calls":                {ID: "0-48", Singular: "Call", Plural: "Calls"},
	"emails":               {ID: "0-49", Singular: "Email", Plural: "Emails"},
	"meetings":             {ID: "0-47", Singular: "Meeting", Plural: "Meetings"},
	"notes":                {ID: "0-46", Singular: "Note", Plural: "Notes"},
	"tasks":                {ID: "0-27", Singular: "Task", Plural: "Tasks"},
	"communications":       {ID: "0-18", Singular: "Communication", Plural: "Communications"},
	"postal_mail":          {ID: "0-116", Singular: "Postal Mail", Plural: "Postal Mails"},
	"leads":                {ID: "0-136", Singular: "Lead", Plural: "Leads"},
	"goals":                {ID: "0-74", Singular: "Goal", Plural: "Goals"},
	"orders":               {ID: "0-123", Singular: "Order", Plural: "Orders"},
	"carts":                {ID: "0-142", Singular: "Cart", Plural: "Carts"},
	"invoices":             {ID: "0-53", Singular: "Invoice", Plural: "Invoices"},
	"feedback_submissions": {ID: "0-19", Singular: "Feedback Submission", Plural: "Feedback Submissions"},
}

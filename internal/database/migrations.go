package database

// migrations is an ordered list of SQL migration groups. Each entry is a slice
// of SQL statements that are executed together in a single transaction. The
// version number is the 1-based index into this slice.
var migrations = [][]string{
	// Migration 1: all core tables
	{
		`CREATE TABLE object_types (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			label_singular TEXT NOT NULL,
			label_plural TEXT NOT NULL,
			primary_display_property TEXT,
			is_custom BOOLEAN NOT NULL DEFAULT FALSE,
			fully_qualified_name TEXT,
			description TEXT,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE property_definitions (
			object_type_id TEXT NOT NULL,
			name TEXT NOT NULL,
			label TEXT NOT NULL,
			type TEXT NOT NULL,
			field_type TEXT NOT NULL,
			group_name TEXT NOT NULL DEFAULT 'contactinformation',
			description TEXT DEFAULT '',
			display_order INTEGER NOT NULL DEFAULT 0,
			has_unique_value BOOLEAN NOT NULL DEFAULT FALSE,
			hidden BOOLEAN NOT NULL DEFAULT FALSE,
			form_field BOOLEAN NOT NULL DEFAULT FALSE,
			calculated BOOLEAN NOT NULL DEFAULT FALSE,
			external_options BOOLEAN NOT NULL DEFAULT FALSE,
			hubspot_defined BOOLEAN NOT NULL DEFAULT FALSE,
			options TEXT,
			calculation_formula TEXT,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (object_type_id, name)
		)`,

		`CREATE TABLE property_groups (
			object_type_id TEXT NOT NULL,
			name TEXT NOT NULL,
			label TEXT NOT NULL,
			display_order INTEGER NOT NULL DEFAULT 0,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			PRIMARY KEY (object_type_id, name)
		)`,

		`CREATE TABLE objects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			object_type_id TEXT NOT NULL,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			archived_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			merged_into_id INTEGER
		)`,
		`CREATE INDEX idx_objects_type ON objects(object_type_id, archived)`,
		`CREATE INDEX idx_objects_type_created ON objects(object_type_id, created_at)`,

		`CREATE TABLE property_values (
			object_id INTEGER NOT NULL,
			property_name TEXT NOT NULL,
			value TEXT,
			updated_at TEXT NOT NULL,
			source TEXT DEFAULT 'API',
			source_id TEXT,
			PRIMARY KEY (object_id, property_name),
			FOREIGN KEY (object_id) REFERENCES objects(id)
		)`,
		`CREATE INDEX idx_property_values_value ON property_values(property_name, value)`,

		`CREATE TABLE property_value_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			object_id INTEGER NOT NULL,
			property_name TEXT NOT NULL,
			value TEXT,
			timestamp TEXT NOT NULL,
			source TEXT DEFAULT 'API',
			source_id TEXT,
			FOREIGN KEY (object_id) REFERENCES objects(id)
		)`,
		`CREATE INDEX idx_prop_history ON property_value_history(object_id, property_name, timestamp)`,

		`CREATE TABLE association_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_object_type TEXT NOT NULL,
			to_object_type TEXT NOT NULL,
			category TEXT NOT NULL,
			label TEXT,
			inverse_label TEXT,
			UNIQUE(from_object_type, to_object_type, category, label)
		)`,

		`CREATE TABLE associations (
			from_object_id INTEGER NOT NULL,
			to_object_id INTEGER NOT NULL,
			association_type_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (from_object_id, to_object_id, association_type_id),
			FOREIGN KEY (from_object_id) REFERENCES objects(id),
			FOREIGN KEY (to_object_id) REFERENCES objects(id),
			FOREIGN KEY (association_type_id) REFERENCES association_types(id)
		)`,
		`CREATE INDEX idx_assoc_from ON associations(from_object_id, association_type_id)`,
		`CREATE INDEX idx_assoc_to ON associations(to_object_id)`,

		`CREATE TABLE pipelines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			object_type_id TEXT NOT NULL,
			label TEXT NOT NULL,
			display_order INTEGER NOT NULL DEFAULT 0,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE pipeline_stages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id INTEGER NOT NULL,
			label TEXT NOT NULL,
			display_order INTEGER NOT NULL DEFAULT 0,
			metadata TEXT DEFAULT '{}',
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (pipeline_id) REFERENCES pipelines(id)
		)`,

		`CREATE TABLE lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			object_type_id TEXT NOT NULL,
			processing_type TEXT NOT NULL,
			processing_status TEXT NOT NULL DEFAULT 'COMPLETE',
			filter_branch TEXT,
			list_version INTEGER NOT NULL DEFAULT 1,
			folder_id INTEGER,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			deleted_at TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE list_memberships (
			list_id INTEGER NOT NULL,
			object_id INTEGER NOT NULL,
			added_at TEXT NOT NULL,
			PRIMARY KEY (list_id, object_id),
			FOREIGN KEY (list_id) REFERENCES lists(id),
			FOREIGN KEY (object_id) REFERENCES objects(id)
		)`,

		`CREATE TABLE imports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			state TEXT NOT NULL DEFAULT 'STARTED',
			source TEXT DEFAULT 'API',
			opt_out_import BOOLEAN NOT NULL DEFAULT FALSE,
			request_json TEXT,
			metadata TEXT DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE import_errors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			import_id INTEGER NOT NULL,
			error_type TEXT NOT NULL,
			error_message TEXT,
			invalid_value TEXT,
			object_type TEXT,
			line_number INTEGER,
			created_at TEXT NOT NULL,
			FOREIGN KEY (import_id) REFERENCES imports(id)
		)`,

		`CREATE TABLE exports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			state TEXT NOT NULL DEFAULT 'ENQUEUED',
			export_type TEXT NOT NULL,
			object_type TEXT NOT NULL,
			object_properties TEXT NOT NULL,
			request_json TEXT,
			result_data BLOB,
			record_count INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE owners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			first_name TEXT,
			last_name TEXT,
			user_id INTEGER,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		`CREATE TABLE request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			request_body TEXT,
			response_body TEXT,
			duration_ms INTEGER,
			correlation_id TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX idx_request_log_time ON request_log(created_at)`,
	},
}

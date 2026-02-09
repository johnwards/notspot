package store

import "database/sql"

// Store holds all sub-stores used by the application.
type Store struct {
	DB      *sql.DB
	Objects ObjectStore
	Search  SearchStore
	Imports ImportStore
	Exports ExportStore
	Owners  OwnerStore
	Lists   ListStore
}

// New creates a Store with all sub-stores initialized.
func New(db *sql.DB) *Store {
	return &Store{
		DB:      db,
		Objects: NewSQLiteObjectStore(db),
		Search:  NewSQLiteSearchStore(db),
		Imports: NewSQLiteImportStore(db),
		Exports: NewSQLiteExportStore(db),
		Owners:  NewSQLiteOwnerStore(db),
		Lists:   NewSQLiteListStore(db),
	}
}

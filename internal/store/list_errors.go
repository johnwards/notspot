package store

import "fmt"

// ErrConflict is returned when a unique constraint is violated.
var ErrConflict = fmt.Errorf("conflict")

// ErrDynamicListMutation is returned when trying to mutate memberships on a DYNAMIC list.
var ErrDynamicListMutation = fmt.Errorf("membership mutations not allowed on DYNAMIC lists")

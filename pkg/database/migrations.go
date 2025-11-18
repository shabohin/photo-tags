package database

import (
	_ "embed"
)

//go:embed migrations/001_initial_schema.sql
var InitialSchema string

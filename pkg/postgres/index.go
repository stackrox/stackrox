package postgres

// IndexDefinition describes an index that should exist on a database table.
// CreateSQL contains the full CREATE INDEX CONCURRENTLY statement, generated at codegen time.
type IndexDefinition struct {
	Name      string
	CreateSQL string
}

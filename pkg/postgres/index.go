package postgres

// IndexDefinition describes an index that should exist on a database table.
// CreateSQL contains the full CREATE INDEX statement, generated at codegen time.
// The Background field determines when the index is created:
//   - Background: false → created at startup by the migrator
//   - Background: true  → created after startup by the background migration runner
type IndexDefinition struct {
	Name       string
	CreateSQL  string
	Background bool
}

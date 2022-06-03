package postgres

// CreateStmts holds the create statements for creating sql table.
type CreateStmts struct {
	Table     string
	Indexes   []string
	GormModel interface{}
	Children  []*CreateStmts
	PostStmts []string
}

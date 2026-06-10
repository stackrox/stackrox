package postgres

// CreateStmts holds the model and statements for creating sql table.
type CreateStmts struct {
	GormModel       any
	Children        []*CreateStmts
	PostStmts       []string
	Partition       bool
	PartitionCreate string
}

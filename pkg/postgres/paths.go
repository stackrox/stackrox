package postgres

// CreateStmts holds the model and statements for creating sql table.
type CreateStmts struct {
	GormModel       interface{}
	Children        []*CreateStmts
	PostStmts       []string
	Partition       bool
	PartitionCreate string
}

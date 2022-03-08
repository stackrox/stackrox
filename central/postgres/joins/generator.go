package joins

import v1 "github.com/stackrox/rox/generated/api/v1"

// Generator provides functionality to get SQL join clauses.
type Generator interface {
	// JoinForCategory returns the tables and where clauses for joining source search category with destination search category.
	JoinForCategory(src, dst v1.SearchCategory) ([]string, []string, error)
	// JoinForSchema returns the tables and where clauses for joining source schema with destination schema.
	JoinForSchema(src, dst string) ([]string, []string, error)
}

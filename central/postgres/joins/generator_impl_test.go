package joins

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

func TestJoinGeneratorForSingleParent(t *testing.T) {
	t.Parallel()

	grandParent := getSchema("grandparent")
	parent := getSchema("parent", grandParent)
	me := getSchema("me", parent)
	child := getSchema("child", me)
	grandChild := getSchema("grandchild", child)

	schemaMap := map[string]*walker.Schema{
		grandParent.Table: grandParent,
		parent.Table:      parent,
		me.Table:          me,
		child.Table:       child,
		grandChild.Table:  grandChild,
	}

	g := newJoinGenerator()
	g.generateJoinsForDBSchema(schemaMap)
	for _, c := range []struct {
		src      string
		dst      string
		expected *sqlJoinClauseParts
	}{
		{
			src: grandParent.Table,
			dst: parent.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandparent", "parent"},
				wheres: []string{"grandparent.id = parent.grandparent_id"},
			},
		},
		{
			src: grandChild.Table,
			dst: child.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandchild", "child"},
				wheres: []string{"grandchild.child_id = child.id"},
			},
		},
		{
			src: grandParent.Table,
			dst: grandChild.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandparent", "parent", "me", "child", "grandchild"},
				wheres: []string{
					"grandparent.id = parent.grandparent_id",
					"parent.id = me.parent_id",
					"me.id = child.me_id",
					"child.id = grandchild.child_id",
				},
			},
		},
		{
			src: grandChild.Table,
			dst: grandParent.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandchild", "child", "me", "parent", "grandparent"},
				wheres: []string{
					"grandchild.child_id = child.id",
					"child.me_id = me.id",
					"me.parent_id = parent.id",
					"parent.grandparent_id = grandparent.id",
				},
			},
		},
		{
			src: me.Table,
			dst: grandChild.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"me", "child", "grandchild"},
				wheres: []string{
					"me.id = child.me_id",
					"child.id = grandchild.child_id",
				},
			},
		},
		{
			src: child.Table,
			dst: parent.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"child", "me", "parent"},
				wheres: []string{
					"child.me_id = me.id",
					"me.parent_id = parent.id",
				},
			},
		},
		{
			src: me.Table,
			dst: me.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"me"},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s -> %s", c.src, c.dst), func(t *testing.T) {
			actualTables, actualWheres, err := g.JoinForSchema(c.src, c.dst)
			assert.NoError(t, err)
			assert.Equal(t, c.expected.tables, actualTables)
			assert.Equal(t, c.expected.wheres, actualWheres)
		})

	}
}

func TestJoinGeneratorForMultipleParents(t *testing.T) {
	t.Parallel()

	father := getSchema("father")
	mother := getSchema("mother")
	child := getSchema("child", father, mother)
	grandSon := getSchema("grandSon", child)
	grandDaughter := getSchema("grandDaughter", child)

	stepFather := getSchema("stepFather")
	stepChild := getSchema("stepChild", stepFather, mother)

	schemaMap := map[string]*walker.Schema{
		father.Table:        father,
		mother.Table:        mother,
		child.Table:         child,
		grandSon.Table:      grandSon,
		grandDaughter.Table: grandDaughter,

		stepFather.Table: stepFather,
		stepChild.Table:  stepChild,
	}

	g := newJoinGenerator()
	g.generateJoinsForDBSchema(schemaMap)
	for _, c := range []struct {
		src      string
		dst      string
		expected *sqlJoinClauseParts
	}{
		{
			src: father.Table,
			dst: grandSon.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"father", "child", "grandSon"},
				wheres: []string{
					"father.id = child.father_id",
					"child.id = grandSon.child_id",
				},
			},
		},
		{
			src: grandSon.Table,
			dst: father.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandSon", "child", "father"},
				wheres: []string{
					"grandSon.child_id = child.id",
					"child.father_id = father.id",
				},
			},
		},
		{
			src: father.Table,
			dst: stepChild.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"father", "child", "mother", "stepChild"},
				wheres: []string{
					"father.id = child.father_id",
					"child.mother_id = mother.id",
					"mother.id = stepChild.mother_id",
				},
			},
		},
		{
			src: stepChild.Table,
			dst: father.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"stepChild", "mother", "child", "father"},
				wheres: []string{
					"stepChild.mother_id = mother.id",
					"mother.id = child.mother_id",
					"child.father_id = father.id",
				},
			},
		},
		{
			src: stepChild.Table,
			dst: grandDaughter.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"stepChild", "mother", "child", "grandDaughter"},
				wheres: []string{
					"stepChild.mother_id = mother.id",
					"mother.id = child.mother_id",
					"child.id = grandDaughter.child_id",
				},
			},
		},
		{
			src: grandDaughter.Table,
			dst: stepFather.Table,
			expected: &sqlJoinClauseParts{
				tables: []string{"grandDaughter", "child", "mother", "stepChild", "stepFather"},
				wheres: []string{
					"grandDaughter.child_id = child.id",
					"child.mother_id = mother.id",
					"mother.id = stepChild.mother_id",
					"stepChild.stepFather_id = stepFather.id",
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s -> %s", c.src, c.dst), func(t *testing.T) {
			actualTables, actualWheres, err := g.JoinForSchema(c.src, c.dst)
			assert.NoError(t, err)
			assert.Equal(t, c.expected.tables, actualTables)
			assert.Equal(t, c.expected.wheres, actualWheres)
		})

	}
}

func getSchema(curr string, parents ...*walker.Schema) *walker.Schema {
	ret := &walker.Schema{
		Table: curr,
		Fields: []walker.Field{
			{
				Name:       "id",
				ColumnName: "id",
				Options: walker.PostgresOptions{
					PrimaryKey: true,
				},
			},
			{
				Name:       "name",
				ColumnName: "name",
			},
		},
	}

	for _, parent := range parents {
		ret.Parents = append(ret.Parents, parent)
		parent.Children = append(parent.Children, ret)
	}
	return ret
}

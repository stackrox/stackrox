package postgres

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFetchConfig(t *testing.T) {
	cfg := defaultFetchConfig()
	assert.True(t, cfg.includeChildren, "default config should include children")
}

func TestApplyFetchOptions(t *testing.T) {
	cases := map[string]struct {
		opts           []FetchOption
		expectChildren bool
	}{
		"no options uses default (include children)": {
			opts:           nil,
			expectChildren: true,
		},
		"WithChildren explicitly includes children": {
			opts:           []FetchOption{WithChildren()},
			expectChildren: true,
		},
		"WithoutChildren excludes children": {
			opts:           []FetchOption{WithoutChildren()},
			expectChildren: false,
		},
		"last option wins": {
			opts:           []FetchOption{WithoutChildren(), WithChildren()},
			expectChildren: true,
		},
		"last option wins reversed": {
			opts:           []FetchOption{WithChildren(), WithoutChildren()},
			expectChildren: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := applyFetchOptions(tc.opts)
			assert.Equal(t, tc.expectChildren, cfg.includeChildren)
		})
	}
}

func TestSchemaForFetch(t *testing.T) {
	store := &noSerializedGenericStore[struct{}]{
		schema: testSchemaWithChildren(),
	}

	t.Run("includeChildren returns original schema", func(t *testing.T) {
		cfg := fetchConfig{includeChildren: true}
		got := store.schemaForFetch(cfg)
		assert.Same(t, store.schema, got)
	})

	t.Run("excludeChildren returns copy without children", func(t *testing.T) {
		cfg := fetchConfig{includeChildren: false}
		got := store.schemaForFetch(cfg)
		assert.NotSame(t, store.schema, got)
		assert.Nil(t, got.Children)
		assert.Equal(t, store.schema.Table, got.Table)
		// Original schema still has children.
		assert.NotEmpty(t, store.schema.Children)
	})
}

func testSchemaWithChildren() *walker.Schema {
	parent := &walker.Schema{
		Table: "parent",
		Fields: []walker.Field{
			{Name: "id", ColumnName: "id"},
		},
	}
	child := &walker.Schema{
		Table:  "parent_children",
		Parent: parent,
	}
	parent.Children = []*walker.Schema{child}
	return parent
}

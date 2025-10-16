package search

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestEmptyQuery(t *testing.T) {
	protoassert.Equal(t, &v1.Query{}, NewQueryBuilder().ProtoQuery())
}

func TestDocIDs(t *testing.T) {
	cases := []struct {
		desc   string
		docIDs []string
	}{
		{
			desc:   "no doc ids",
			docIDs: []string{},
		},
		{
			desc:   "one doc id",
			docIDs: []string{"1"},
		},
		{
			desc:   "two doc ids",
			docIDs: []string{"1", "2"},
		},
	}
	for _, c := range cases {
		q := NewQueryBuilder().AddDocIDs(c.docIDs...).ProtoQuery()
		expected := v1.Query_builder{
			BaseQuery: v1.BaseQuery_builder{
				DocIdQuery: v1.DocIDQuery_builder{
					Ids: c.docIDs,
				}.Build(),
			}.Build(),
		}.Build()
		protoassert.Equal(t, expected, q)
	}
}

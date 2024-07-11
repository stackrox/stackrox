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
		expected := &v1.Query{
			Query: &v1.Query_BaseQuery{
				BaseQuery: &v1.BaseQuery{
					Query: &v1.BaseQuery_DocIdQuery{
						DocIdQuery: &v1.DocIDQuery{
							Ids: c.docIDs,
						},
					},
				},
			},
		}
		protoassert.Equal(t, expected, q)
	}
}

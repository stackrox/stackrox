package blevesearch

import (
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/pkg/set"
)

type hasField interface {
	Field() string
}

func getAllFieldPaths(q query.Query) set.StringSet {
	s := set.NewStringSet()
	switch subQ := q.(type) {
	case *query.ConjunctionQuery:
		for _, c := range subQ.Conjuncts {
			s = s.Union(getAllFieldPaths(c))
		}
		return s
	case *query.DisjunctionQuery:
		for _, c := range subQ.Disjuncts {
			s = s.Union(getAllFieldPaths(c))
		}
		return s
	}
	if fieldInterface, ok := q.(hasField); ok {
		s.Add(fieldInterface.Field())
	}
	return s
}

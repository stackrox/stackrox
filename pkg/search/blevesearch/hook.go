package blevesearch

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

// ResultsFilterFunc is a function for filtering Bleve search results.
type ResultsFilterFunc func([]*search.DocumentMatch) ([]*search.DocumentMatch, error)

// Hook can be used to customize bleve search behavior. It is strictly more powerful than wrapping the
// `RunSearchRequest` function, since it can also apply to subqueries.
type Hook struct {
	InternalHighlightFields []string
	SubQueryHooks           HookForCategory
	ResultsFilter           ResultsFilterFunc
}

// HookForCategory allows obtaining a bleve search hook depending on the category.
type HookForCategory func(v1.SearchCategory) *Hook

func (h *Hook) apply(highlightCtx highlightContext) *hookPostProcessor {
	if h.ResultsFilter == nil {
		return nil
	}

	pp := &hookPostProcessor{
		Hook: h,
	}
	for _, field := range h.InternalHighlightFields {
		if _, ok := highlightCtx[field]; !ok {
			highlightCtx.AddFieldToHighlight(field)
			pp.internalHighlightFields = append(pp.internalHighlightFields, field)
		}
	}
	return pp
}

type hookPostProcessor struct {
	*Hook
	internalHighlightFields []string
	highlightCtx            highlightContext
}

func (pp *hookPostProcessor) apply(result *bleve.SearchResult) (*bleve.SearchResult, error) {
	numOldHits := len(result.Hits)
	newHits, err := pp.ResultsFilter(result.Hits)
	if err != nil {
		return nil, err
	}

	maxScore := 0.0
	for i, hit := range newHits {
		if hit.Score > maxScore {
			maxScore = hit.Score
		}
		for _, field := range pp.internalHighlightFields {
			delete(hit.Fields, field)
			delete(hit.FieldArrayPositions, field)
		}
		hit.HitNumber = uint64(i)
	}

	result.Total -= uint64(numOldHits - len(newHits))
	result.MaxScore = maxScore
	result.Hits = newHits

	for _, field := range pp.internalHighlightFields {
		delete(pp.highlightCtx, field)
	}

	return result, nil
}

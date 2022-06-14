package blevesearch

import (
	"math"
	"reflect"

	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/index"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/scorer"
	"github.com/blevesearch/bleve/size"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/search/blevesearch/validpositions"
)

const (
	defaultTerm = "term"
)

var reflectStaticSizeNegationSearcher int

func init() {
	var bs NegationSearcher
	reflectStaticSizeNegationSearcher = int(reflect.TypeOf(bs).Size())
}

// NegationSearcher is the negated form of a search query
type NegationSearcher struct {
	indexReader       index.IndexReader
	typeSearcher      search.Searcher
	negationSearcher  search.Searcher
	queryNorm         float64
	currTypeMatch     *search.DocumentMatch
	currNegationMatch *search.DocumentMatch
	currentID         index.IndexInternalID
	scorer            *scorer.ConjunctionQueryScorer
	matches           []*search.DocumentMatch
	initialized       bool
	done              bool
	required          bool
}

// NewNegationSearcher creates a new negation searcher
func NewNegationSearcher(indexReader index.IndexReader, typeSearcher search.Searcher, mustNotSearcher search.Searcher, options search.SearcherOptions, required bool) (*NegationSearcher, error) {
	// build our searcher
	rv := NegationSearcher{
		indexReader:      indexReader,
		typeSearcher:     typeSearcher,
		negationSearcher: mustNotSearcher,
		scorer:           scorer.NewConjunctionQueryScorer(options),
		matches:          make([]*search.DocumentMatch, 2),
		required:         required,
	}
	rv.computeQueryNorm()
	return &rv, nil
}

// Size returns the size of the searcher
func (s *NegationSearcher) Size() int {
	sizeInBytes := reflectStaticSizeNegationSearcher + size.SizeOfPtr
	sizeInBytes += s.typeSearcher.Size()
	sizeInBytes += s.negationSearcher.Size()

	for _, entry := range s.matches {
		if entry != nil {
			sizeInBytes += entry.Size()
		}
	}

	return sizeInBytes
}

func (s *NegationSearcher) computeQueryNorm() {
	// first calculate sum of squared weights
	sumOfSquaredWeights := s.typeSearcher.Weight()

	// now compute query norm from this
	s.queryNorm = 1.0 / math.Sqrt(sumOfSquaredWeights)
	// finally tell all the downstream searchers the norm
	s.typeSearcher.SetQueryNorm(s.queryNorm)
}

func (s *NegationSearcher) initSearchers(ctx *search.SearchContext) error {
	var err error
	// get all searchers pointing at their first match
	if s.currTypeMatch != nil {
		ctx.DocumentMatchPool.Put(s.currTypeMatch)
	}
	s.currTypeMatch, err = s.typeSearcher.Next(ctx)
	if err != nil {
		return err
	}

	if s.currNegationMatch != nil {
		ctx.DocumentMatchPool.Put(s.currNegationMatch)
	}
	s.currNegationMatch, err = s.negationSearcher.Next(ctx)
	if err != nil {
		return err
	}

	if s.currTypeMatch != nil {
		s.currentID = s.currTypeMatch.IndexInternalID
	} else {
		s.currentID = nil
	}

	s.initialized = true
	return nil
}

func (s *NegationSearcher) advanceNextMust(ctx *search.SearchContext, skipReturn *search.DocumentMatch) error {
	var err error

	if s.currTypeMatch != skipReturn {
		ctx.DocumentMatchPool.Put(s.currTypeMatch)
	}
	s.currTypeMatch, err = s.typeSearcher.Next(ctx)
	if err != nil {
		return err
	}

	if s.currTypeMatch != nil {
		s.currentID = s.currTypeMatch.IndexInternalID
	} else {
		s.currentID = nil
	}
	return nil
}

// Weight sets the weight for the searcher
func (s *NegationSearcher) Weight() float64 {
	return s.typeSearcher.Weight()
}

// SetQueryNorm sets the query norm for the searcher
func (s *NegationSearcher) SetQueryNorm(qnorm float64) {
	s.typeSearcher.SetQueryNorm(qnorm)
}

func (s *NegationSearcher) getDocumentFromReader(id index.IndexInternalID) (*document.Document, error) {
	eid, err := s.indexReader.ExternalID(id)
	if err != nil {
		return nil, err
	}
	return s.indexReader.Document(eid)
}

type fieldRef struct {
	name               string
	fields             []document.Field
	fieldTermLocations []search.FieldTermLocation
}

func handleFieldAndFieldTermLocations(ref *fieldRef) ([]search.FieldTermLocation, bool) {
	var (
		ftls          []search.FieldTermLocation
		completeMatch = true
	)

	// Create a tree from the field term locations and then verify that it contains all of the array positions within
	// the fields
	tree := validpositions.NewTree()
	for _, ftl := range ref.fieldTermLocations {
		tree.Add(ftl.Location.ArrayPositions)
	}

	for _, field := range ref.fields {
		// If a field was not in the term then it was not a complete match and build the field term location from that
		if !tree.Contains(field.ArrayPositions()) {
			completeMatch = false
			ftls = append(ftls, fieldToFieldTermLocation(field))
		}
	}
	return ftls, completeMatch
}

func (s *NegationSearcher) shouldExclude(dm *search.DocumentMatch) ([]search.FieldTermLocation, bool, error) {
	var newFieldTermLocations []search.FieldTermLocation
	if !s.required {
		return newFieldTermLocations, true, nil
	}

	// Get the internal document so the fields are available
	internalDoc, err := s.getDocumentFromReader(dm.IndexInternalID)
	if err != nil {
		return nil, false, err
	}

	// Build a reference map that contains both the field term locations and all of the fields of that path
	fieldRefMap := make(map[string]*fieldRef)
	for _, ftl := range dm.FieldTermLocations {
		if _, ok := fieldRefMap[ftl.Field]; !ok {
			fieldRefMap[ftl.Field] = &fieldRef{
				name: ftl.Field,
			}
		}
		fieldRefMap[ftl.Field].fieldTermLocations = append(fieldRefMap[ftl.Field].fieldTermLocations, ftl)
	}

	for _, field := range internalDoc.Fields {
		ref := fieldRefMap[field.Name()]
		if ref == nil {
			continue
		}
		ref.fields = append(ref.fields, field)
	}

	completeMatch := true
	var fieldTermLocations []search.FieldTermLocation
	for _, ref := range fieldRefMap {
		// Take the reference and get the field term locations as well as if it was a complete match
		ftls, match := handleFieldAndFieldTermLocations(ref)
		completeMatch = completeMatch && match
		fieldTermLocations = append(fieldTermLocations, ftls...)
	}
	return fieldTermLocations, completeMatch, nil
}

func fieldToFieldTermLocation(field document.Field) search.FieldTermLocation {
	return search.FieldTermLocation{
		Term:  defaultTerm,
		Field: field.Name(),
		Location: search.Location{
			ArrayPositions: field.ArrayPositions(),
		},
	}
}

func (s *NegationSearcher) convertFieldsToFieldToLocations(id index.IndexInternalID) ([]search.FieldTermLocation, error) {
	doc, err := s.getDocumentFromReader(id)
	if err != nil {
		return nil, err
	}

	newFieldTermLocations := make([]search.FieldTermLocation, 0, len(doc.Fields))
	for _, field := range doc.Fields {
		newFieldTermLocations = append(newFieldTermLocations, fieldToFieldTermLocation(field))
	}
	return newFieldTermLocations, nil
}

// Next finds the next match for the searcher
func (s *NegationSearcher) Next(ctx *search.SearchContext) (*search.DocumentMatch, error) {
	if s.done {
		return nil, nil
	}

	if !s.initialized {
		err := s.initSearchers(ctx)
		if err != nil {
			return nil, err
		}
	}

	var (
		err     error
		exclude bool
		rv      *search.DocumentMatch
	)

	for s.currentID != nil {
		var (
			newFieldTermLocations []search.FieldTermLocation
			matchedMustNot        bool
		)
		if s.currNegationMatch != nil {
			cmp := s.currNegationMatch.IndexInternalID.Compare(s.currentID)
			switch {
			case cmp < 0:
				ctx.DocumentMatchPool.Put(s.currNegationMatch)
				// advance must not searcher to our candidate entry
				s.currNegationMatch, err = s.negationSearcher.Advance(ctx, s.currentID)
				if err != nil {
					return nil, err
				}
				if s.currNegationMatch == nil || !s.currNegationMatch.IndexInternalID.Equals(s.currentID) {
					break
				}
				fallthrough
			case cmp == 0:
				newFieldTermLocations, exclude, err = s.shouldExclude(s.currNegationMatch)
				if err != nil {
					return nil, err
				}
				if exclude {
					// the candidate is excluded
					err = s.advanceNextMust(ctx, nil)
					if err != nil {
						return nil, err
					}
					continue
				}
				matchedMustNot = true
			}
		}

		// If the above logic did not match the must not clause then we need to set the fieldTermLocations to the fields of the document
		if !matchedMustNot {
			newFieldTermLocations, err = s.convertFieldsToFieldToLocations(s.currentID)
			if err != nil {
				return nil, err
			}
		}

		s.currTypeMatch.FieldTermLocations = newFieldTermLocations

		// match is OK anyway
		cons := s.matches[0:1]
		cons[0] = s.currTypeMatch
		rv = s.scorer.Score(ctx, cons)
		err = s.advanceNextMust(ctx, rv)
		if err != nil {
			return nil, err
		}
		break
	}

	if rv == nil {
		s.done = true
	}

	return rv, nil
}

// Advance advances the searcher
func (s *NegationSearcher) Advance(ctx *search.SearchContext, ID index.IndexInternalID) (*search.DocumentMatch, error) {
	if s.done {
		return nil, nil
	}

	if !s.initialized {
		err := s.initSearchers(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Advance the searcher only if the cursor is trailing the lookup ID
	if s.currentID == nil || s.currentID.Compare(ID) < 0 {
		var err error
		if s.currTypeMatch != nil {
			ctx.DocumentMatchPool.Put(s.currTypeMatch)
		}
		s.currTypeMatch, err = s.typeSearcher.Advance(ctx, ID)
		if err != nil {
			return nil, err
		}

		if s.currNegationMatch == nil || s.currNegationMatch.IndexInternalID.Compare(ID) < 0 {
			if s.currNegationMatch != nil {
				ctx.DocumentMatchPool.Put(s.currNegationMatch)
			}
			s.currNegationMatch, err = s.negationSearcher.Advance(ctx, ID)
			if err != nil {
				return nil, err
			}
		}

		if s.currTypeMatch != nil {
			s.currentID = s.currTypeMatch.IndexInternalID
		} else {
			s.currentID = nil
		}
	}

	return s.Next(ctx)
}

// Count returns the total count
func (s *NegationSearcher) Count() uint64 {
	// for now return a worst case
	return s.typeSearcher.Count()
}

// Min returns the min of the searcher
func (s *NegationSearcher) Min() int {
	return 0
}

// Close closes the searchers
func (s *NegationSearcher) Close() error {
	el := errorhelpers.NewErrorList("Closing searchers")
	el.AddError(s.typeSearcher.Close())
	el.AddError(s.negationSearcher.Close())
	return el.ToError()
}

// DocumentMatchPoolSize helps determine how much memory is needed
func (s *NegationSearcher) DocumentMatchPoolSize() int {
	rv := 2
	rv += s.typeSearcher.DocumentMatchPoolSize()
	rv += s.negationSearcher.DocumentMatchPoolSize()
	return rv
}

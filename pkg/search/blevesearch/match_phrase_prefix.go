//  Copyright (c) 2014 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blevesearch

import (
	"fmt"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"
)

// MatchPhrasePrefixQuery implements a network query
type MatchPhrasePrefixQuery struct {
	MatchPhrasePrefix string `json:"match_phrase_prefix"`
	FieldVal          string `json:"field,omitempty"`
	Analyzer          string `json:"analyzer,omitempty"`
}

// NewMatchPhrasePrefixQuery creates a new Query
// for matching phrase prefix in the index.
// An Analyzer is chosen based on the field.
// Input text is analyzed using this analyzer.
// Token terms resulting from this analysis are
// used to build a search phrase.  Result documents
// must match this phrase prefix. Queried field must have been indexed with
// IncludeTermVectors set to true.
func NewMatchPhrasePrefixQuery(field, matchPhrasePrefix string) *MatchPhrasePrefixQuery {
	return &MatchPhrasePrefixQuery{
		MatchPhrasePrefix: matchPhrasePrefix,
		FieldVal:          field,
	}
}

// SetField sets the field value
func (q *MatchPhrasePrefixQuery) SetField(f string) {
	q.FieldVal = f
}

// Field returns the field value
func (q *MatchPhrasePrefixQuery) Field() string {
	return q.FieldVal
}

// Searcher implements the search interface
func (q *MatchPhrasePrefixQuery) Searcher(i index.IndexReader, m mapping.IndexMapping, options search.SearcherOptions) (search.Searcher, error) {
	field := q.FieldVal
	if q.FieldVal == "" {
		field = m.DefaultSearchField()
	}

	analyzerName := ""
	if q.Analyzer != "" {
		analyzerName = q.Analyzer
	} else {
		analyzerName = m.AnalyzerNameForPath(field)
	}
	analyzer := m.AnalyzerNamed(analyzerName)
	if analyzer == nil {
		return nil, fmt.Errorf("no analyzer named '%s' registered", q.Analyzer)
	}

	tokens := analyzer.Analyze([]byte(q.MatchPhrasePrefix))
	if len(tokens) > 0 {
		phrase := tokenStreamToPhrase(tokens)
		if len(phrase) > 0 {
			// expand tokens at last position to terms from dictionary
			var terms []string
			for _, prefix := range phrase[len(phrase)-1] {
				fieldDict, err := i.FieldDictPrefix(field, []byte(prefix))
				if err != nil {
					return nil, err
				}
				tfd, err := fieldDict.Next()
				for err == nil && tfd != nil {
					terms = append(terms, tfd.Term)
					tfd, err = fieldDict.Next()
				}
			}
			if len(terms) > 0 {
				phrase[len(phrase)-1] = terms
			}
		}
		phraseQuery := query.NewMultiPhraseQuery(phrase, field)
		return phraseQuery.Searcher(i, m, options)
	}

	noneQuery := query.NewMatchNoneQuery()
	return noneQuery.Searcher(i, m, options)
}

func tokenStreamToPhrase(tokens analysis.TokenStream) [][]string {
	firstPosition := int(^uint(0) >> 1)
	lastPosition := 0
	for _, token := range tokens {
		if token.Position < firstPosition {
			firstPosition = token.Position
		}
		if token.Position > lastPosition {
			lastPosition = token.Position
		}
	}
	phraseLen := lastPosition - firstPosition + 1
	if phraseLen > 0 {
		rv := make([][]string, phraseLen)
		for _, token := range tokens {
			pos := token.Position - firstPosition
			rv[pos] = append(rv[pos], string(token.Term))
		}
		return rv
	}
	return nil
}

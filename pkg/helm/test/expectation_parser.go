package test

import (
	"bufio"
	"strings"
	"unicode"

	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
)

// parseExpectations parses an "expect" section. The expect section consists of several jq filters, one per line.
// In order to allow longer filter expressions, a filter expression may be continued on the next line. This is indicated
// by having the continuation line start with any whitespace character.
func parseExpectations(spec string) ([]*gojq.Query, error) {
	var queries []*gojq.Query
	scanner := bufio.NewScanner(strings.NewReader(spec))
	current := ""
	scanned := true
	for lineNo := 1; scanned; lineNo++ {
		scanned = scanner.Scan()
		var next string
		if scanned {
			line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
			trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
			if len(trimmed) < len(line) {
				// Continuation line.
				if current == "" {
					return nil, errors.Errorf("unexpected continuation on line %d", lineNo)
				}
				current += " " + trimmed
				continue
			}
			next = line
		}

		if current != "" {
			query, err := gojqParse(current)
			if err != nil {
				return nil, errors.Wrapf(err, "parsing query ending on line %d", lineNo-1)
			}
			queries = append(queries, query)
		}
		current = next
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "parsing expectations")
	}

	return queries, nil
}

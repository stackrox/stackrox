package search

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// ParseRawQuery takes the text based query and converts to the ParsedSearchRequest proto
func ParseRawQuery(query string) (*v1.ParsedSearchRequest, error) {
	if query == "" {
		return nil, errors.New("Query cannot be empty")
	}

	pairs := strings.Split(query, "+")
	parsedRequest := &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
	}

	for _, pair := range pairs {
		key, values, valid := parsePair(pair)
		if !valid {
			continue
		}

		if added, err := addStringQuery(parsedRequest, key, values); err != nil {
			return nil, err
		} else if added {
			continue
		}

		valuesSlice := strings.Split(values, ",")

		if err := addGeneralField(parsedRequest, key, valuesSlice); err != nil {
			return nil, err
		}
	}

	if len(parsedRequest.GetFields()) == 0 && parsedRequest.GetStringQuery() == "" {
		return nil, errors.New("After parsing, query is empty")
	}
	return parsedRequest, nil
}

func parsePair(pair string) (key string, values string, valid bool) {
	pair = strings.TrimSpace(pair)
	if len(pair) == 0 {
		return
	}

	spl := strings.SplitN(pair, ":", 2)
	// len < 2 implies there isn't a colon and the second check verifies that the : wasn't the last char
	if len(spl) < 2 || spl[1] == "" {
		return
	}
	return spl[0], spl[1], true
}

func addStringQuery(request *v1.ParsedSearchRequest, key, value string) (added bool, err error) {
	// Check if its a raw query
	if strings.EqualFold(key, "has") {
		if request.GetStringQuery() != "" {
			err = fmt.Errorf("There can only be 1 raw string query")
			return
		}
		added = true
		request.StringQuery = value
	}
	return
}

func addGeneralField(request *v1.ParsedSearchRequest, key string, values []string) error {
	// transform the key into its mapped form
	if _, ok := request.Fields[key]; !ok {
		request.Fields[key] = &v1.ParsedSearchRequest_Values{}
	}

	// Append the fields < key: [value value] >
	request.Fields[key].Values = append(request.Fields[key].Values, values...)
	return nil
}

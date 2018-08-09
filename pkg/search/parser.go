package search

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// QueryParser parses queries
// besides the standard scopes, and string queries.
type QueryParser struct{}

// ParseRawQuery takes the text based query and converts to the ParsedSearchRequest proto
func (q *QueryParser) ParseRawQuery(query string) (*v1.ParsedSearchRequest, error) {
	if query == "" {
		return nil, errors.New("Query cannot be empty")
	}

	pairs := strings.Split(query, "+")
	parsedRequest := &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
	}

	var scopeFields = map[string][]string{
		"Cluster":     {},
		"Namespace":   {},
		"Label Key":   {},
		"Label Value": {},
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
		if added := addScopeField(scopeFields, key, valuesSlice); added {
			continue
		}

		if err := q.addGeneralField(parsedRequest, key, valuesSlice); err != nil {
			return nil, err
		}
	}

	// Populate Scope query
	//////////////////////////////////////////////////////////////////
	scopes := populateLabelKeys(scopeFields["Label Key"])
	scopes = populateLabelValues(scopes, scopeFields["Label Value"])
	scopes = populateNamespaces(scopes, scopeFields["Namespace"])
	scopes = populateClusters(scopes, scopeFields["Cluster"])

	parsedRequest.Scopes = make([]*v1.Scope, 0, len(scopes))
	for _, scope := range scopes {
		diffScope := cloneScope(scope)
		parsedRequest.Scopes = append(parsedRequest.Scopes, diffScope)
	}
	//////////////////////////////////////////////////////////////////

	if len(parsedRequest.GetScopes()) == 0 && len(parsedRequest.GetFields()) == 0 && parsedRequest.GetStringQuery() == "" {
		return nil, errors.New("After parsing, query is empty")
	}
	return parsedRequest, nil
}

func populateNamespaces(scopes []*v1.Scope, namespaces []string) []*v1.Scope {
	if len(namespaces) == 0 {
		return scopes
	}
	if len(scopes) == 0 {
		newScopes := make([]*v1.Scope, 0, len(namespaces))
		for _, namespace := range namespaces {
			newScopes = append(newScopes, &v1.Scope{Namespace: namespace})
		}
		return newScopes
	}
	newScopes := make([]*v1.Scope, 0, len(scopes)*len(namespaces))
	for _, scope := range scopes {
		for _, namespace := range namespaces {
			newScope := cloneScope(scope)
			newScope.Namespace = namespace
			newScopes = append(newScopes, newScope)
		}
	}
	return newScopes
}

func populateClusters(scopes []*v1.Scope, clusters []string) []*v1.Scope {
	if len(clusters) == 0 {
		return scopes
	}
	if len(scopes) == 0 {
		newScopes := make([]*v1.Scope, 0, len(clusters))
		for _, cluster := range clusters {
			newScopes = append(newScopes, &v1.Scope{Cluster: cluster})
		}
		return newScopes
	}
	newScopes := make([]*v1.Scope, 0, len(scopes)*len(clusters))
	for _, scope := range scopes {
		for _, cluster := range clusters {
			newScope := cloneScope(scope)
			newScope.Cluster = cluster
			newScopes = append(newScopes, newScope)
		}
	}
	return newScopes
}

func populateLabelKeys(keys []string) []*v1.Scope {
	scopes := make([]*v1.Scope, 0, len(keys))
	for _, key := range keys {
		scope := new(v1.Scope)
		scope.Label = &v1.Scope_Label{
			Key: key,
		}
		scopes = append(scopes, &v1.Scope{
			Label: &v1.Scope_Label{
				Key: key,
			},
		})
	}
	return scopes
}

func populateLabelValues(scopes []*v1.Scope, values []string) []*v1.Scope {
	if len(values) == 0 {
		return scopes
	}

	if len(scopes) == 0 {
		newScopes := make([]*v1.Scope, 0, len(values))
		for _, value := range values {
			newScopes = append(newScopes, &v1.Scope{Label: &v1.Scope_Label{Value: value}})
		}
		return newScopes
	}

	newScopes := make([]*v1.Scope, 0, len(scopes)*len(values))
	for _, scope := range scopes {
		for _, value := range values {
			newScope := cloneScope(scope)
			if scope.GetLabel() == nil {
				newScope.Label = &v1.Scope_Label{
					Value: value,
				}
			} else {
				newScope.Label.Value = value
			}
			newScopes = append(newScopes, newScope)
		}
	}
	return newScopes
}

func cloneScope(s *v1.Scope) (scope *v1.Scope) {
	scope = new(v1.Scope)
	scope.Cluster = s.GetCluster()
	scope.Namespace = s.GetNamespace()
	if s.GetLabel() != nil {
		scope.Label = &v1.Scope_Label{
			Key:   s.GetLabel().GetKey(),
			Value: s.GetLabel().GetValue(),
		}
	}
	return
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

func addScopeField(scopeFields map[string][]string, key string, values []string) bool {
	// if value is a scope field, then added it to the scope fields which are mapped separately
	if vals, ok := scopeFields[key]; ok {
		scopeFields[key] = append(vals, values...)
		return true
	}
	return false
}

func (q *QueryParser) addGeneralField(request *v1.ParsedSearchRequest, key string, values []string) error {
	// transform the key into its mapped form
	if _, ok := request.Fields[key]; !ok {
		request.Fields[key] = &v1.ParsedSearchRequest_Values{}
	}

	// Append the fields < key: [value value] >
	request.Fields[key].Values = append(request.Fields[key].Values, values...)
	return nil
}

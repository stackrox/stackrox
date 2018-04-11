package search

import (
	"errors"
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func populateNamespaces(scopes []v1.Scope, namespaces []string) []v1.Scope {
	if len(namespaces) == 0 {
		return scopes
	}
	if len(scopes) == 0 {
		newScopes := make([]v1.Scope, 0, len(namespaces))
		for _, namespace := range namespaces {
			newScopes = append(newScopes, v1.Scope{Namespace: namespace})
		}
		return newScopes
	}
	newScopes := make([]v1.Scope, 0, len(scopes)*len(namespaces))
	for _, scope := range scopes {
		tmpScope := scope
		for _, namespace := range namespaces {
			tmpScope.Namespace = namespace
			newScopes = append(newScopes, tmpScope)
		}
	}
	return newScopes
}

func populateClusters(scopes []v1.Scope, clusters []string) []v1.Scope {
	if len(clusters) == 0 {
		return scopes
	}
	if len(scopes) == 0 {
		newScopes := make([]v1.Scope, 0, len(clusters))
		for _, cluster := range clusters {
			newScopes = append(newScopes, v1.Scope{Cluster: cluster})
		}
		return newScopes
	}
	newScopes := make([]v1.Scope, 0, len(scopes)*len(clusters))
	for _, scope := range scopes {
		tmpScope := scope
		for _, cluster := range clusters {
			tmpScope.Cluster = cluster
			newScopes = append(newScopes, tmpScope)
		}
	}
	return newScopes
}

func populateLabels(labels []string) (scopes []v1.Scope, err error) {
	for _, label := range labels {
		var scope v1.Scope
		values := strings.Split(label, "=")
		if len(values) != 2 {
			err = fmt.Errorf("Labels must container an '=' between the key and value: %s", label)
			return
		}
		scope.Label = &v1.Scope_Label{
			Key:   values[0],
			Value: values[1],
		}
		scopes = append(scopes, scope)
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

func addGeneralField(request *v1.ParsedSearchRequest, key string, values []string) error {
	// transform the key into its mapped form
	transformedKey, ok := allOptionsMaps[key]
	if !ok {
		return fmt.Errorf("Key %s is not a valid search option", key)
	}
	if _, ok := request.Fields[transformedKey]; !ok {
		request.Fields[transformedKey] = new(v1.ParsedSearchRequest_Values)
	}
	// Append the fields < key: [value value] >
	request.Fields[transformedKey].Values = append(request.Fields[transformedKey].Values, values...)
	return nil
}

// ParseRawQuery takes the text based query and converts to the ParsedSearchRequest proto
func ParseRawQuery(query string) (*v1.ParsedSearchRequest, error) {
	if query == "" {
		return nil, errors.New("Query cannot be empty")
	}
	pairs := strings.Split(query, "+")
	parsedRequest := &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
	}
	var scopeFields = map[string][]string{
		"Cluster":   {},
		"Namespace": {},
		"Label":     {},
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
		if err := addGeneralField(parsedRequest, key, valuesSlice); err != nil {
			return nil, err
		}
	}
	// Compute the cross product of the scopes
	scopes, err := populateLabels(scopeFields["Label"])
	if err != nil {
		return nil, err
	}
	scopes = populateNamespaces(scopes, scopeFields["Namespace"])
	scopes = populateClusters(scopes, scopeFields["Cluster"])
	parsedRequest.Scopes = make([]*v1.Scope, 0, len(scopes))

	for _, scope := range scopes {
		diffScope := scope
		parsedRequest.Scopes = append(parsedRequest.Scopes, &diffScope)
	}

	if len(parsedRequest.GetScopes()) == 0 && len(parsedRequest.GetFields()) == 0 && parsedRequest.GetStringQuery() == "" {
		return nil, errors.New("After parsing, query is empty")
	}
	return parsedRequest, nil
}

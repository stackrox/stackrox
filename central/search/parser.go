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

// ParseRawQuery takes the text based query and converts to the ParsedSearchRequest proto
func ParseRawQuery(request *v1.RawSearchRequest) (*v1.ParsedSearchRequest, error) {
	if request.GetQuery() == "" {
		return nil, errors.New("Query cannot be empty")
	}

	pairs := strings.Split(request.GetQuery(), " ")
	parsedRequest := &v1.ParsedSearchRequest{
		Fields: make(map[string]*v1.ParsedSearchRequest_Values),
	}

	var scopeFields = map[string][]string{
		"cluster":   {},
		"namespace": {},
		"label":     {},
	}

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if len(pair) == 0 {
			continue
		}
		values := strings.Split(pair, ":")
		if len(values) == 1 {
			if parsedRequest.GetStringQuery() != "" {
				return nil, fmt.Errorf("There can only be 1 raw string query")
			}
			parsedRequest.StringQuery = values[0]
			continue
		} else if len(values) != 2 {
			return nil, fmt.Errorf("Extra colon was found in '%s', but they are not allowed in search strings", pair)
		}
		k, v := values[0], values[1]
		if vals, ok := scopeFields[k]; ok {
			scopeFields[k] = append(vals, v)
			continue
		}
		if _, ok := parsedRequest.Fields[k]; !ok {
			parsedRequest.Fields[k] = new(v1.ParsedSearchRequest_Values)
		}
		parsedRequest.Fields[k].Values = append(parsedRequest.Fields[k].Values, v)
	}
	scopes, err := populateLabels(scopeFields["label"])
	if err != nil {
		return nil, err
	}
	scopes = populateNamespaces(scopes, scopeFields["namespace"])
	scopes = populateClusters(scopes, scopeFields["cluster"])
	if len(scopes) == 0 {
		return parsedRequest, nil
	}
	parsedRequest.Scopes = make([]*v1.Scope, 0, len(scopes))
	for _, scope := range scopes {
		diffScope := scope
		parsedRequest.Scopes = append(parsedRequest.Scopes, &diffScope)
	}
	return parsedRequest, nil
}

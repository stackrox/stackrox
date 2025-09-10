package permissions

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/centralproxy/pkg/rbac"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// GraphQLQueryAnalyzer analyzes GraphQL queries to determine required permissions
type GraphQLQueryAnalyzer struct{}

// NewGraphQLQueryAnalyzer creates a new GraphQL query analyzer
func NewGraphQLQueryAnalyzer() *GraphQLQueryAnalyzer {
	return &GraphQLQueryAnalyzer{}
}

// ExtractPermissions analyzes a GraphQL query and returns the required permissions
func (a *GraphQLQueryAnalyzer) ExtractPermissions(queryBody []byte) ([]rbac.Permission, error) {
	var request struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := json.Unmarshal(queryBody, &request); err != nil {
		return nil, errors.Wrap(err, "failed to parse GraphQL request")
	}

	return a.analyzeQuery(request.Query)
}

// analyzeQuery performs simple string-based analysis of GraphQL queries
// TODO: Implement proper GraphQL AST parsing for production use
func (a *GraphQLQueryAnalyzer) analyzeQuery(query string) ([]rbac.Permission, error) {
	var permissions []rbac.Permission
	permissionSet := make(map[rbac.Permission]bool)

	// Simple string-based field detection
	// In production, this should use a proper GraphQL parser
	fields := a.extractFields(query)

	for _, field := range fields {
		if perms, exists := rbac.VirtualGVRMapping[field]; exists {
			for _, perm := range perms {
				if !permissionSet[perm] {
					permissions = append(permissions, perm)
					permissionSet[perm] = true
				}
			}
		}
	}

	// If no specific fields found, default to basic image permissions
	if len(permissions) == 0 {
		log.Debug("No specific fields detected, using default permissions")
		permissions = append(permissions, rbac.Permission{
			Resource: "images",
			Verb:     "list",
		})
	}

	log.Debugf("Extracted permissions from GraphQL query: %+v", permissions)
	return permissions, nil
}

// extractFields performs simple field extraction from GraphQL query string
// This is a basic implementation - production should use proper GraphQL parsing
func (a *GraphQLQueryAnalyzer) extractFields(query string) []string {
	var fields []string
	
	// Remove comments and normalize whitespace
	query = a.cleanQuery(query)

	// Look for known field patterns
	knownFields := []string{
		"images",
		"vulnerabilities", 
		"imageVulnerabilities",
		"policies",
		"violations",
		"deployments",
		"clusters",
		"nodes",
	}

	for _, field := range knownFields {
		if strings.Contains(query, field) {
			fields = append(fields, field)
		}
	}

	// Special case: if vulnerabilities is nested under images, use imageVulnerabilities mapping
	if strings.Contains(query, "images") && strings.Contains(query, "vulnerabilities") {
		// Check if vulnerabilities appears to be nested under images
		if a.isNestedField(query, "images", "vulnerabilities") {
			fields = append(fields, "imageVulnerabilities")
		}
	}

	return fields
}

// cleanQuery removes comments and normalizes whitespace
func (a *GraphQLQueryAnalyzer) cleanQuery(query string) string {
	// Remove comments
	lines := strings.Split(query, "\n")
	var cleanLines []string
	for _, line := range lines {
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		cleanLines = append(cleanLines, line)
	}
	
	cleaned := strings.Join(cleanLines, " ")
	
	// Normalize whitespace
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}
	
	return strings.TrimSpace(cleaned)
}

// isNestedField checks if childField appears to be nested under parentField
func (a *GraphQLQueryAnalyzer) isNestedField(query, parentField, childField string) bool {
	// Simple heuristic: if childField appears after parentField and there's a '{' in between
	parentIdx := strings.Index(query, parentField)
	childIdx := strings.Index(query, childField)
	
	if parentIdx == -1 || childIdx == -1 || childIdx <= parentIdx {
		return false
	}
	
	// Check if there's an opening brace between parent and child
	between := query[parentIdx:childIdx]
	return strings.Contains(between, "{")
}

// GetFieldPermissions returns the permissions required for a specific GraphQL field
func GetFieldPermissions(field string) []rbac.Permission {
	if perms, exists := rbac.VirtualGVRMapping[field]; exists {
		return perms
	}
	
	// Default permission if field not recognized
	return []rbac.Permission{
		{Resource: "images", Verb: "list"},
	}
}
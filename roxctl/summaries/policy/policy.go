package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

//go:generate genny -in=../../../pkg/set/generic.go -out=policy-set.go -pkg policy gen "KeyType=*storage.Policy"

// TotalPolicyAmountKey relates to the key within the Policy summary map which yields the total amount of violated
// policies
const TotalPolicyAmountKey = "TOTAL"

// Severity is used for easier comparing the prettified string version of storage.Severity
// when sorting policies by severity.
type Severity int

const (
	// LowSeverity represents a "LOW" Severity for policies
	LowSeverity Severity = iota
	// MediumSeverity represents a "MEDIUM" Severity for policies
	MediumSeverity
	// HighSeverity represents a "HIGH" Severity for policies
	HighSeverity
	// CriticalSeverity represents a "CRITICAL" Severity for policies
	CriticalSeverity
)

func (s Severity) String() string {
	return [...]string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}[s]
}

func policySeverityFromString(s string) Severity {
	switch s {
	case LowSeverity.String():
		return LowSeverity
	case MediumSeverity.String():
		return MediumSeverity
	case HighSeverity.String():
		return HighSeverity
	case CriticalSeverity.String():
		return CriticalSeverity
	default:
		return 0
	}
}

// NewPolicySummaryForPrinting creates a Result that shall be used for printing and holds
// all relevant information regarding violated policies, failing policies and a summary of all violated policies
// by severity
// NOTE: The returned *Result CAN be passed to json.Marshal
func NewPolicySummaryForPrinting(alerts []*storage.Alert, forbiddenEnforcementAction storage.EnforcementAction) *Result {
	entityMetadataMap := createEntityMetadataMap(alerts)
	numOfSeveritiesByEntities := createNumOfSeverityByEntity(entityMetadataMap)
	numOfSeveritiesAcrossEntities := createNumOfSeverityMap()
	policiesByEntity := make(map[string]map[string]*Policy, len(entityMetadataMap))
	breakingPoliciesByEntities := make(map[string]*StoragePolicySet, len(entityMetadataMap))

	for _, alert := range alerts {
		entityID := getEntityIDFromAlert(alert)
		// create StoragePolicySet for failed policies if it does not yet exist for the
		// current entity
		if _, exists := breakingPoliciesByEntities[entityID]; !exists {
			set := NewStoragePolicySet()
			breakingPoliciesByEntities[entityID] = &set
		}
		// create map for policies if it does not yet exist for the current entity
		if _, exists := policiesByEntity[entityID]; !exists {
			policiesByEntity[entityID] = map[string]*Policy{}
		}

		p := alert.GetPolicy()
		policyID := p.GetId()
		_, exists := policiesByEntity[entityID][policyID]
		// do not add the Policy again to the map, since multiple alerts
		// can point to the same Policy. Instead, we need to merge the violation messages
		// of the alerts, since the violation could be different.
		if exists {
			policyJSON := policiesByEntity[entityID][policyID]
			policyJSON.Violation += getAlertViolationsString(alert)
			// we can skip here, since we do not want to add the Policy either
			// to the overall set (duplicate) or to the failing set (duplicate)
			continue
		}

		strippedPolicySeverityEnum := trimSeverityEnumSuffix(p.GetSeverity())
		policiesByEntity[entityID][policyID] = &Policy{
			Name:        p.GetName(),
			Severity:    strippedPolicySeverityEnum,
			Description: p.GetDescription(),
			Remediation: p.GetRemediation(),
			Violation:   getAlertViolationsString(alert),
		}
		// if the Policy has a forbidden storage.EnforcementAction add it to the set of
		// failing policies if it is not yet added. Need to check this since
		// multiple alerts can point to the same Policy
		if checkIfPolicyHasForbiddenEnforcementAction(p, forbiddenEnforcementAction) &&
			!breakingPoliciesByEntities[entityID].Contains(p) {
			breakingPoliciesByEntities[entityID].Add(p)
		}

		// increase the severity count & total account for the entity and the total amount
		numOfSeveritiesByEntities[entityID][strippedPolicySeverityEnum]++
		numOfSeveritiesByEntities[entityID][TotalPolicyAmountKey]++
		numOfSeveritiesAcrossEntities[strippedPolicySeverityEnum]++
		numOfSeveritiesAcrossEntities[TotalPolicyAmountKey]++
	}

	resultsForEntities := createResultsForEntities(entityMetadataMap, policiesByEntity, breakingPoliciesByEntities,
		numOfSeveritiesByEntities)

	return &Result{
		Results: resultsForEntities,
		Summary: numOfSeveritiesAcrossEntities,
	}
}

// createResultsForEntities will create a EntityResult for each entity and add the corresponding violated
// policies, breaking policies and number of severities to it
func createResultsForEntities(entityMetadataMap map[string]EntityMetadata,
	policiesByEntities map[string]map[string]*Policy,
	breakingPoliciesSetByEntities map[string]*StoragePolicySet,
	numOfSeverityByEntities map[string]map[string]int) []EntityResult {

	sortedEntitiesMetadata := sortMetadataByEntity(getEntityMetadataFromMap(entityMetadataMap))
	resultsForEntities := make([]EntityResult, 0, len(sortedEntitiesMetadata))

	for _, metadata := range sortedEntitiesMetadata {
		entityResult := EntityResult{
			Metadata:         metadata,
			Summary:          numOfSeverityByEntities[metadata.ID],
			ViolatedPolicies: sortPoliciesBySeverity(getPoliciesFromMap(policiesByEntities[metadata.ID])),
			BreakingPolicies: sortBreakingPoliciesByName(getBreakingPolicies(
				breakingPoliciesSetByEntities[metadata.ID].AsSlice())),
		}
		resultsForEntities = append(resultsForEntities, entityResult)
	}
	return resultsForEntities
}

// createEntityMetadataMap creates a map of EntityMetadata where the entity ID is the key
func createEntityMetadataMap(alerts []*storage.Alert) map[string]EntityMetadata {
	var result = map[string]EntityMetadata{}
	for _, alert := range alerts {
		var additionalInfo = map[string]string{}
		entityID := getEntityIDFromAlert(alert)
		switch entity := alert.Entity.(type) {
		case *storage.Alert_Deployment_:
			if _, exists := result[entityID]; !exists {
				additionalInfo["name"] = entity.Deployment.Name
				additionalInfo["type"] = entity.Deployment.Type
				additionalInfo["namespace"] = entity.Deployment.Namespace
				result[entityID] = EntityMetadata{AdditionalInfo: additionalInfo, ID: entityID}
			}
		case *storage.Alert_Image:
			if _, exists := result[entityID]; !exists {
				additionalInfo["name"] = entity.Image.Name.GetFullName()
				additionalInfo["type"] = "image"
				result[entityID] = EntityMetadata{AdditionalInfo: additionalInfo, ID: entityID}
			}
		default:
			// this should theoretically not happen, this means that an unknown entity is specified.
			// the returned entityID will be "unkown"
			result[entityID] = EntityMetadata{ID: entityID}
		}
	}
	return result
}

// createNumOfSeverityByEntity creates a map where for each entity a summary map
// created by createNumOfSeverityMap is included
func createNumOfSeverityByEntity(resultMetadata map[string]EntityMetadata) map[string]map[string]int {
	var numOfSeverityByEntity = make(map[string]map[string]int, len(resultMetadata))
	for entityID := range resultMetadata {
		numOfSeverityByEntity[entityID] = createNumOfSeverityMap()
	}
	return numOfSeverityByEntity
}

// createNumOfSeverityMap creates a map that holds all trimmed severity enums
// and total amount as keys
func createNumOfSeverityMap() map[string]int {
	numOfSeverityMap := make(map[string]int, 5)
	numOfSeverityMap[TotalPolicyAmountKey] = 0
	numOfSeverityMap[trimSeverityEnumSuffix(storage.Severity_LOW_SEVERITY)] = 0
	numOfSeverityMap[trimSeverityEnumSuffix(storage.Severity_MEDIUM_SEVERITY)] = 0
	numOfSeverityMap[trimSeverityEnumSuffix(storage.Severity_HIGH_SEVERITY)] = 0
	numOfSeverityMap[trimSeverityEnumSuffix(storage.Severity_CRITICAL_SEVERITY)] = 0
	return numOfSeverityMap
}

// getEntityMetadataFromMap returns an array of EntityMetadata from all map values
func getEntityMetadataFromMap(m map[string]EntityMetadata) []EntityMetadata {
	result := make([]EntityMetadata, 0, len(m))
	for _, metadata := range m {
		result = append(result, metadata)
	}
	return result
}

// getEntityIDFromAlert retrieves the entity ID based on the alert's entity
func getEntityIDFromAlert(alert *storage.Alert) string {
	switch entity := alert.Entity.(type) {
	case *storage.Alert_Deployment_:
		return entity.Deployment.GetId()
	case *storage.Alert_Image:
		return entity.Image.Name.GetFullName()
	}
	// return an "unkown" id opposed to an empty string; we still create the report, but the metadata
	// will be empty
	return "unknown"
}

// getPoliciesFromMap returns an array of Policy from all map values
func getPoliciesFromMap(policyMap map[string]*Policy) []Policy {
	policies := make([]Policy, 0, len(policyMap))
	for _, policy := range policyMap {
		policies = append(policies, *policy)
	}
	return policies
}

// getAlertViolationsString merges all violation messages of an alert
func getAlertViolationsString(alert *storage.Alert) string {
	var res string
	for _, violation := range alert.GetViolations() {
		res += fmt.Sprintf("- %s\n", violation.Message)
	}
	return res
}

// getBreakingPolicies creates BreakingPolicy representation of breaking policies
func getBreakingPolicies(breakingPolicies []*storage.Policy) []BreakingPolicy {
	breakingPoliciesJSON := make([]BreakingPolicy, 0, len(breakingPolicies))
	for _, policy := range breakingPolicies {
		breakingPoliciesJSON = append(breakingPoliciesJSON, BreakingPolicy{
			Name:        policy.GetName(),
			Remediation: policy.GetRemediation(),
		})
	}
	return breakingPoliciesJSON
}

// checkIfPolicyHasForbiddenEnforcementAction iterates through the Policy's enforcement actions and returns true
// if the forbidden action is included
func checkIfPolicyHasForbiddenEnforcementAction(policy *storage.Policy, forbiddenAction storage.EnforcementAction) bool {
	for _, action := range policy.GetEnforcementActions() {
		if action == forbiddenAction {
			return true
		}
	}
	return false
}

// trimSeverityEnumSuffix trims the proto generated "_SEVERITY" suffix
func trimSeverityEnumSuffix(severity storage.Severity) string {
	return strings.TrimSuffix(severity.String(), "_SEVERITY")
}

// sortBreakingPoliciesByName sorts an array of BreakingPolicy by name
func sortBreakingPoliciesByName(breakingPolicies []BreakingPolicy) []BreakingPolicy {
	sort.SliceStable(breakingPolicies, func(i, j int) bool {
		return breakingPolicies[i].Name < breakingPolicies[j].Name
	})
	return breakingPolicies
}

// sortPoliciesBySeverity sorts policies by their Severity from highest (CriticalSeverity) to lowest (LowSeverity)
func sortPoliciesBySeverity(policies []Policy) []Policy {
	// sort alphabetically by name first
	sort.SliceStable(policies, func(i, j int) bool {
		return policies[i].Name < policies[j].Name
	})
	// sort decreasing by severity, CRITICAL being highest - LOW being lowest
	sort.SliceStable(policies, func(i, j int) bool {
		return policySeverityFromString(policies[i].Severity) > policySeverityFromString(policies[j].Severity)
	})
	return policies
}

// sortMetadataByEntity sorts EntityMetadata by their type's name and by name
func sortMetadataByEntity(metadata []EntityMetadata) []EntityMetadata {
	sort.SliceStable(metadata, func(i, j int) bool {
		if metadata[i].AdditionalInfo["type"] < metadata[j].AdditionalInfo["type"] {
			return true
		}
		if metadata[i].AdditionalInfo["type"] > metadata[j].AdditionalInfo["type"] {
			return false
		}
		return metadata[i].AdditionalInfo["name"] < metadata[j].AdditionalInfo["name"]
	})
	return metadata
}

// Result represents a summary of found violated policies on an entity basis (entity being either an image or a deployment)
type Result struct {
	Results []EntityResult `json:"results,omitempty"`
	Summary map[string]int `json:"summary,omitempty"`
}

// GetTotalAmountOfBreakingPolicies calculates the amount of breaking policies for all EntityResult
func (r *Result) GetTotalAmountOfBreakingPolicies() int {
	amount := 0
	for _, entityResult := range r.Results {
		amount += len(entityResult.BreakingPolicies)
	}
	return amount
}

// GetResultNames retrieves a list of the names for all results
func (r *Result) GetResultNames() []string {
	var names []string

	for _, entityResult := range r.Results {
		names = append(names, entityResult.Metadata.GetName())
	}

	return names
}

// EntityResult represents a result consisting of policies for a specific entity
type EntityResult struct {
	Metadata         EntityMetadata   `json:"metadata"`
	Summary          map[string]int   `json:"summary"`
	ViolatedPolicies []Policy         `json:"violatedPolicies,omitempty"`
	BreakingPolicies []BreakingPolicy `json:"breakingPolicies,omitempty"`
}

// Policy represents information about a policy
type Policy struct {
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Violation   string `json:"violation"`
	Remediation string `json:"remediation"`
}

// BreakingPolicy represents information about a breaking policy
type BreakingPolicy struct {
	Name        string `json:"name"`
	Remediation string `json:"remediation"`
}

// EntityMetadata provides information about the entity associated with the policy results
type EntityMetadata struct {
	ID             string            `json:"id"`
	AdditionalInfo map[string]string `json:"additionalInfo"`
}

// GetName retrieves the name of the EntityMetadata
func (e *EntityMetadata) GetName() string {
	return e.AdditionalInfo["name"]
}

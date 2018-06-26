package mappings

import (
	deploymentMappings "bitbucket.org/stack-rox/apollo/central/deployment/index/mappings"
	policyMappings "bitbucket.org/stack-rox/apollo/central/policy/index/mappings"
	"bitbucket.org/stack-rox/apollo/pkg/search/blevesearch"
	"github.com/blevesearch/bleve/mapping"
)

// DocumentMap is the document mapping for alerts.
var DocumentMap = func() *mapping.DocumentMapping {
	alertMap := blevesearch.FieldsToDocumentMapping(OptionsMap)
	blevesearch.AddDefaultTypeField(alertMap)

	alertMap.Properties["alert"].AddSubDocumentMapping("deployment", deploymentMappings.DocumentMap.Properties["deployment"])
	alertMap.Properties["alert"].AddSubDocumentMapping("policy", policyMappings.DocumentMap.Properties["policy"])
	return alertMap
}()

package resolvers

import (
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()

	utils.Must(schema.AddUnionType("EmbeddedVulnerabilityVectors", []string{
		"CVSSV2",
		"CVSSV3",
	}))
}

// EmbeddedVulnerabilityVectorsResolver resolves to one of two version of CVSS data.
type EmbeddedVulnerabilityVectorsResolver struct {
	resolver interface{}
}

// ToCVSSV2 returns the vector as a CVSSV2 data object.
func (resolver *EmbeddedVulnerabilityVectorsResolver) ToCVSSV2() (*cVSSV2Resolver, bool) {
	res, ok := resolver.resolver.(*cVSSV2Resolver)
	return res, ok
}

// ToCVSSV3 returns the vector as a CVSSV3 data object.
func (resolver *EmbeddedVulnerabilityVectorsResolver) ToCVSSV3() (*cVSSV3Resolver, bool) {
	res, ok := resolver.resolver.(*cVSSV3Resolver)
	return res, ok
}

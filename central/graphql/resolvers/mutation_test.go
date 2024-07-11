package resolvers

import (
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

// knownMutations is a complete list of mutations currently in use by the GraphQL API
// The test in this file was added to deter anyone in the future from adding new mutations.
// GraphQL mutation operations were determined to not add a significant amount of value compared to RESTful API
// endpoints, and expansion of their usage creates complicated engineering problems that could be avoided by simply
// streamlining mutation operations through gRPC (ex. automated audit logging).
// Think before adding new values to this list.
var knownMutations = set.NewFrozenStringSet(
	"approveVulnerabilityRequest",
	"complianceTriggerRuns",
	"deferVulnerability",
	"deleteVulnerabilityRequest",
	"denyVulnerabilityRequest",
	"markVulnerabilityFalsePositive",
	"undoVulnerabilityRequest",
	"updateVulnerabilityRequest",
)

func TestMutation(t *testing.T) {
	suite.Run(t, new(MutationTestSuite))
}

type MutationTestSuite struct {
	suite.Suite

	schema *graphql.Schema
}

func (s *MutationTestSuite) SetupTest() {
	var err error
	s.schema, err = graphql.ParseSchema(Schema(), &Resolver{})
	s.NoError(err)
}

// TestKnownMutation tests to ensure that the GraphQL schema does not contain any new mutations
func (s *MutationTestSuite) TestKnownMutation() {
	for _, v := range s.schema.ASTSchema().Objects {
		if v.Name == "Mutation" {
			for _, f := range v.Fields {
				if !knownMutations.Contains(f.Name) {
					s.Failf("Unknown mutation in GraphQL schema", "mutation name %q", f.Name)
				}
			}

			// only one mutation object in the schema, so we can short circuit here
			return
		}
	}
}

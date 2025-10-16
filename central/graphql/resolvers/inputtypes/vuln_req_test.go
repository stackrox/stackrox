package inputtypes

import (
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestVulnReqInputResolvers(t *testing.T) {
	suite.Run(t, new(VulnReqInputResolversTestSuite))
}

type VulnReqInputResolversTestSuite struct {
	suite.Suite
}

func (s *VulnReqInputResolversTestSuite) TestAsRequestExpiry() {
	now := time.Now()
	var cases = []struct {
		name           string
		input          *VulnReqExpiry
		expectedExpiry *storage.RequestExpiry
	}{
		{
			name: "Expiring at time",
			input: &VulnReqExpiry{
				ExpiresWhenFixed: boolPtr(false),
				ExpiresOn:        &graphql.Time{Time: now},
			},
			expectedExpiry: storage.RequestExpiry_builder{
				ExpiresOn: proto.ValueOrDefault(protoconv.ConvertTimeToTimestamp(now)),
			}.Build(),
		},
		{
			name:  "Expiring at time with nil ExpiresWhenFixed",
			input: &VulnReqExpiry{ExpiresOn: &graphql.Time{Time: now}},
			expectedExpiry: storage.RequestExpiry_builder{
				ExpiresOn: proto.ValueOrDefault(protoconv.ConvertTimeToTimestamp(now)),
			}.Build(),
		},
		{
			name: "Expiring when fixed with some value in ExpiresOn",
			input: &VulnReqExpiry{
				ExpiresWhenFixed: boolPtr(true),
				ExpiresOn:        &graphql.Time{Time: now},
			},
			expectedExpiry: storage.RequestExpiry_builder{
				ExpiresWhenFixed: proto.Bool(true),
			}.Build(),
		},
		{
			name:  "Expiring when fixed with nil ExpiresOn",
			input: &VulnReqExpiry{ExpiresWhenFixed: boolPtr(true)},
			expectedExpiry: storage.RequestExpiry_builder{
				ExpiresWhenFixed: proto.Bool(true),
			}.Build(),
		},
		{
			name: "Never expiring with nil ExpiresOn",
			input: &VulnReqExpiry{
				ExpiresWhenFixed: boolPtr(false),
				ExpiresOn:        nil,
			},
			expectedExpiry: &storage.RequestExpiry{},
		},
		{
			name:           "Never expiring with nil VulnReqExpiry",
			input:          nil,
			expectedExpiry: &storage.RequestExpiry{},
		},
		{
			name:           "Never expiring with empty VulnReqExpiry",
			input:          &VulnReqExpiry{},
			expectedExpiry: &storage.RequestExpiry{},
		},
		{
			name: "Expiring at zero time",
			input: &VulnReqExpiry{
				ExpiresWhenFixed: boolPtr(false),
				ExpiresOn:        &graphql.Time{Time: time.Time{}},
			},
			expectedExpiry: storage.RequestExpiry_builder{
				ExpiresOn: proto.ValueOrDefault(protoconv.ConvertTimeToTimestamp(time.Time{})),
			}.Build(),
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.True(c.expectedExpiry.EqualVT(c.input.AsRequestExpiry()))
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

package scannerv4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	s4ClientMocks "github.com/stackrox/rox/pkg/scannerv4/client/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetVulnDefinitionsInfo(t *testing.T) {
	errExpected := true
	var noErr error
	var noMetadata *v4.Metadata
	tmsFromTime, _ := protocompat.ConvertTimeToTimestampOrError(time.Time{})
	testCases := []struct {
		desc         string
		clientRet    *v4.Metadata
		clientRetErr error
		errExpected  bool
	}{
		{
			"error when client returns error",
			noMetadata, errors.New("fake"), errExpected,
		},
		{
			"error when client returns nil metadata",
			noMetadata, noErr, errExpected,
		},
		{
			"error when client returns nil last vuln update",
			&v4.Metadata{}, noErr, errExpected,
		},
		{
			"error when client returns zero last vuln update",
			&v4.Metadata{LastVulnerabilityUpdate: protocompat.GetProtoTimestampZero()}, noErr, errExpected,
		},
		{
			"error when client returns zero last vuln update (from time)",
			&v4.Metadata{LastVulnerabilityUpdate: tmsFromTime}, noErr, errExpected,
		},
		{
			"success when client returns valid last vuln update",
			&v4.Metadata{LastVulnerabilityUpdate: protocompat.TimestampNow()}, noErr, !errExpected,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			scannerClient := s4ClientMocks.NewMockScanner(ctrl)
			scannerClient.EXPECT().GetMatcherMetadata(gomock.Any()).Return(tc.clientRet, tc.clientRetErr)
			s := scannerv4{scannerClient: scannerClient}

			vdi, err := s.GetVulnDefinitionsInfo()
			if tc.errExpected {
				if tc.clientRetErr != nil {
					require.ErrorContains(t, err, tc.clientRetErr.Error())
				} else {
					require.ErrorContains(t, err, "timestamp")
				}
				assert.Nil(t, vdi)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.clientRet.GetLastVulnerabilityUpdate().AsTime(), vdi.GetLastUpdatedTimestamp().AsTime())
			}
		})
	}

}

// FIXME: This is for dev purposes only - needs a unit test
func TestNodeIndexer(t *testing.T) {
	t.Run("Test node", func(t *testing.T) {
		o := []client.Option{
			client.WithMatcherAddress(":8443"),
			client.SkipTLSVerification,
		}
		scannerClient, err := client.NewGRPCScanner(context.Background(), o...)
		require.NoError(t, err)
		s := scannerv4{scannerClient: scannerClient}
		n := &storage.Node{Name: "Testnode"}

		vr, err := s.GetNodeVulnerabilityReport(n, createIndexReport())
		require.NoError(t, err)
		require.NotNil(t, vr)

		reportJSON, err := json.MarshalIndent(vr, "", "  ")
		require.NoError(t, err)

		log.Info(string(reportJSON))
	})
}

func createIndexReport() *v4.IndexReport {
	ir := &v4.IndexReport{
		HashId:  fmt.Sprintf("sha256:%s", strings.Repeat("a", 64)),
		Success: true,
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "0",
					Name:    "openssh-clients",
					Version: "8.7p1-38.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "openssh",
						Version: "8.7p1-38.el9",
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
				{
					Id:      "1",
					Name:    "skopeo",
					Version: "2:1.14.4-2.rhaos4.16.el9",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "skopeo",
						Version: "2:1.14.4-2.rhaos4.16.el9",
						Kind:    "source",
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:072a75d1b9b36457751ef05031fd69615f21ebaa935c30d74d827328b78fa694|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "0",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
				{
					Id:   "1",
					Name: "cpe:/a:redhat:openshift:4.16::el9",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:a:redhat:openshift:4.16:*:el9:*:*:*:*:*",
				},
			},
			Environments: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{"0", "1"},
				},
			},
			}},
		},
	}
	log.Info("Generating Node IndexReport")
	return ir
}

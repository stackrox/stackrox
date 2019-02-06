package docker

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditCheck(t *testing.T) {
	cases := []struct {
		name   string
		file   *compliance.File
		status framework.Status
	}{
		{
			name: "CIS_Docker_v1_1_0:1_5",
			file: &compliance.File{
				Path:    auditFile,
				Content: []byte("/usr/bin/docker.service\n/usr/bin/docker"),
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:1_5",
			file: &compliance.File{
				Path:    auditFile,
				Content: []byte("/etc/default/docker"),
			},
			status: framework.FailStatus,
		},
	}

	for _, cIt := range cases {
		c := cIt
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()
			registry := framework.RegistrySingleton()
			check := registry.Lookup(c.name)
			require.NotNil(t, check)

			testCluster := &storage.Cluster{
				Id: uuid.NewV4().String(),
			}
			testNodes := []*storage.Node{
				{
					Id:   "A",
					Name: "A",
				},
				{
					Id:   "B",
					Name: "B",
				},
			}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			domain := framework.NewComplianceDomain(testCluster, testNodes, nil)
			data := mocks.NewMockComplianceDataRepository(mockCtrl)

			allFiles := map[string]*compliance.File{
				"/usr/bin/docker.service": {Path: "/usr/bin/docker.service"},
				"/usr/bin/docker":         {Path: "/usr/bin/docker"},
				"/etc/default/docker":     {Path: "/etc/default/docker"},
				c.file.Path:               c.file,
			}

			data.EXPECT().HostScraped().AnyTimes().Return(map[string]*compliance.ComplianceReturn{
				"A": {
					Files: allFiles,
				},
				"B": {
					Files: allFiles,
				},
			})

			run, err := framework.NewComplianceRun(check)
			require.NoError(t, err)
			err = run.Run(context.Background(), domain, data)
			require.NoError(t, err)

			results := run.GetAllResults()
			checkResults := results[c.name]
			require.NotNil(t, checkResults)

			require.Len(t, checkResults.Evidence(), 0)
			for _, node := range domain.Nodes() {
				nodeResults := checkResults.ForChild(node)
				require.NoError(t, nodeResults.Error())
				require.Len(t, nodeResults.Evidence(), 1)
				assert.Equal(t, c.status, nodeResults.Evidence()[0].Status)
			}
		})
	}
}

package docker

import (
	"context"
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

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			registry := framework.RegistrySingleton()
			check := registry.Lookup(c.name)
			require.NotNil(t, check)

			testCluster := &storage.Cluster{
				Id: uuid.NewV4().String(),
			}
			testNodes := []*storage.Node{
				{
					Id: "A",
				},
				{
					Id: "B",
				},
			}

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			domain := framework.NewComplianceDomain(testCluster, testNodes, nil)
			data := mocks.NewMockComplianceDataRepository(mockCtrl)

			data.EXPECT().HostScraped().AnyTimes().Return(map[string]*compliance.ComplianceReturn{
				"A": {
					Files: map[string]*compliance.File{
						c.file.Path: c.file,
					},
				},
				"B": {
					Files: map[string]*compliance.File{
						c.file.Path: c.file,
					},
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

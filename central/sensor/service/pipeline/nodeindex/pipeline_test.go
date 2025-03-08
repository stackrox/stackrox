package nodeindex

import (
	"context"
	"testing"

	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPipelineWithEmptyIndex(t *testing.T) {
	t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	p := &pipelineImpl{}
	expectedError := "unexpected resource type"

	err := p.Run(context.Background(), "", nil, nil)

	assert.Contains(t, err.Error(), expectedError)
}

func TestPipelineWithIncorrectAction(t *testing.T) {
	t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	p := &pipelineImpl{}
	msg := createMsg()
	msg.GetEvent().Action = central.ResourceAction_REMOVE_RESOURCE

	err := p.Run(context.Background(), "", msg, nil)

	assert.Nil(t, err)
}

func TestPipelineEnrichesAndUpserts(t *testing.T) {
	t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
	t.Setenv(features.ScannerV4.EnvVar(), "true")
	node := storage.Node{
		Id: "1",
	}
	ctrl := gomock.NewController(t)
	clusterStore := clusterDatastoreMocks.NewMockDataStore(ctrl)
	nodeDatastore := nodeDatastoreMocks.NewMockDataStore(ctrl)
	nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq("1")).Times(1).Return(&node, true, nil)
	riskManager := riskManagerMocks.NewMockManager(ctrl)
	riskManager.EXPECT().CalculateRiskAndUpsertNode(gomock.Any()).Times(1).Return(nil)
	enricher := nodesEnricherMocks.NewMockNodeEnricher(ctrl)
	enricher.EXPECT().EnrichNodeWithVulnerabilities(gomock.Any(), nil, gomock.Any()).Times(1).Return(nil)

	p := &pipelineImpl{
		clusterStore:  clusterStore,
		nodeDatastore: nodeDatastore,
		riskManager:   riskManager,
		enricher:      enricher,
	}
	msg := createMsg()
	msg.GetEvent().Action = central.ResourceAction_UNSET_ACTION_RESOURCE

	err := p.Run(context.Background(), "", msg, nil)

	assert.Nil(t, err)
}

func createMsg() *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: "1",
				Resource: &central.SensorEvent_IndexReport{
					IndexReport: createIndexReport(),
				},
			},
		},
	}
}

func createIndexReport() *v4.IndexReport {
	return &v4.IndexReport{
		HashId:  "",
		State:   "7", // IndexFinished
		Success: true,
		Err:     "",
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
}

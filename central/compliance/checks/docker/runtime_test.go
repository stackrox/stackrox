package docker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/framework/mocks"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRuntimeChecks(t *testing.T) {
	cases := []struct {
		name      string
		container types.ContainerJSON
		status    framework.Status
	}{
		{
			name: "CIS_Docker_v1_1_0:5_25",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{"hello"},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_25",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{"hello", "no-new-privileges"},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_1",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					AppArmorProfile: "default",
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_1",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					AppArmorProfile: "unconfined",
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_13",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{}},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_13",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0"}},
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name:   "CIS_Docker_v1_1_0:5_29",
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_29",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]struct{}{
						"bridge": {},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_3",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						CapAdd: []string{"CAP_SYS_ADMIN"},
					},
				},
			},
			status: framework.NoteStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_3",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						CapAdd: []string{},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_24",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CgroupParent: "docker",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_24",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CgroupParent: "random",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_11",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CPUShares: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_11",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CPUShares: 10,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: nil,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: &types.Health{
							Status: "",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: &types.Health{
							Status: "yay",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_17",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Devices: []container.DeviceMapping{},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_17",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Devices: []container.DeviceMapping{
								{
									PathOnHost: "/dev/sda",
								},
							},
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_16",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						IpcMode: container.IpcMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_16",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						IpcMode: container.IpcMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_10",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Memory: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_10",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Memory: 100,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_19",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Propagation: mount.PropagationShared,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_19",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Propagation: mount.PropagationPrivate,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_8",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{}},
						},
					},
				},
			},
			status: framework.NoteStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_31",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/var/run/types.sock",
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_31",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/etc/passwd",
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_15",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						PidMode: container.PidMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_15",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						PidMode: container.PidMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_28",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							PidsLimit: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_28",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							PidsLimit: 10,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_4",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Privileged: false,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_4",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Privileged: true,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_7",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostPort: "1025"}},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_7",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostPort: "80"}},
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_12",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						ReadonlyRootfs: true,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_12",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						ReadonlyRootfs: false,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_14",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						RestartPolicy: container.RestartPolicy{
							Name:              "on-failure",
							MaximumRetryCount: 5,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_14",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						RestartPolicy: container.RestartPolicy{
							Name:              "lol",
							MaximumRetryCount: 5,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_21",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"seccomp:unconfined",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_21",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"seccomp:default",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_2",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_2",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"selinux",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_5",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/etc",
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_5",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/opt",
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_9",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						NetworkMode: container.NetworkMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_9",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						NetworkMode: container.NetworkMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_18",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Ulimits: []*units.Ulimit{},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_18",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Ulimits: []*units.Ulimit{
								{
									Name: "abc",
									Soft: 10,
									Hard: 10,
								},
							},
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_30",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UsernsMode: container.UsernsMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_30",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UsernsMode: container.UsernsMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_20",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UTSMode: container.UTSMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_20",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UTSMode: container.UTSMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "stackrox",
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "root",
				},
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

			// Must set the containers to running
			if c.container.ContainerJSONBase != nil {
				if c.container.ContainerJSONBase.State == nil {
					c.container.State = &types.ContainerState{
						Running: true,
					}
				} else {
					c.container.State.Running = true
				}
			} else {
				c.container.ContainerJSONBase = &types.ContainerJSONBase{
					State: &types.ContainerState{
						Running: true,
					},
				}
			}
			if c.container.HostConfig == nil {
				c.container.HostConfig = &types.HostConfig{}
			}

			jsonData, err := json.Marshal(&types.Data{
				Containers: []types.ContainerJSON{
					c.container,
				},
			})
			require.NoError(t, err)

			var jsonDataGZ bytes.Buffer
			gzWriter := gzip.NewWriter(&jsonDataGZ)
			_, err = gzWriter.Write(jsonData)
			require.NoError(t, err)
			require.NoError(t, gzWriter.Close())

			data.EXPECT().HostScraped(nodeNameMatcher("A")).AnyTimes().Return(&compliance.ComplianceReturn{
				DockerData: &compliance.GZIPDataChunk{Gzip: jsonDataGZ.Bytes()},
			})
			data.EXPECT().HostScraped(nodeNameMatcher("B")).AnyTimes().Return(&compliance.ComplianceReturn{
				DockerData: &compliance.GZIPDataChunk{Gzip: jsonDataGZ.Bytes()},
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

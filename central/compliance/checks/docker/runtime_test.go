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
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRuntimeChecks(t *testing.T) {
	cases := []struct {
		name      string
		container docker.ContainerJSON
		status    framework.Status
	}{
		{
			name: "CIS_Docker_v1_1_0:5_25",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						SecurityOpt: []string{"hello"},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_25",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						SecurityOpt: []string{"hello", "no-new-privileges"},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_1",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					AppArmorProfile: "default",
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_1",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					AppArmorProfile: "unconfined",
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_13",
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					NetworkSettingsBase: docker.NetworkSettingsBase{
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
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					NetworkSettingsBase: docker.NetworkSettingsBase{
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
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					Networks: map[string]struct{}{
						"bridge": {},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_3",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						CapAdd: []string{"CAP_SYS_ADMIN"},
					},
				},
			},
			status: framework.NoteStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_3",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						CapAdd: []string{},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_24",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							CgroupParent: "docker",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_24",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							CgroupParent: "random",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_11",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							CPUShares: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_11",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							CPUShares: 10,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					State: &docker.ContainerState{
						Health: nil,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					State: &docker.ContainerState{
						Health: &docker.Health{
							Status: "",
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_26",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					State: &docker.ContainerState{
						Health: &docker.Health{
							Status: "yay",
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_17",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							Devices: []container.DeviceMapping{},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_17",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						IpcMode: container.IpcMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_16",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						IpcMode: container.IpcMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_10",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							Memory: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_10",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							Memory: 100,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_19",
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Propagation: mount.PropagationShared,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_19",
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Propagation: mount.PropagationPrivate,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_8",
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					NetworkSettingsBase: docker.NetworkSettingsBase{
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
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Source: "/var/run/docker.sock",
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_31",
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Source: "/etc/passwd",
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_15",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						PidMode: container.PidMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_15",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						PidMode: container.PidMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_28",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							PidsLimit: 0,
						},
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_28",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							PidsLimit: 10,
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_4",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Privileged: false,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_4",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Privileged: true,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_7",
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					NetworkSettingsBase: docker.NetworkSettingsBase{
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
			container: docker.ContainerJSON{
				NetworkSettings: &docker.NetworkSettings{
					NetworkSettingsBase: docker.NetworkSettingsBase{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						ReadonlyRootfs: true,
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_12",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						ReadonlyRootfs: false,
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_14",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
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
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Source: "/etc",
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_5",
			container: docker.ContainerJSON{
				Mounts: []docker.MountPoint{
					{
						Source: "/opt",
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_9",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						NetworkMode: container.NetworkMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_9",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						NetworkMode: container.NetworkMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_18",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
							Ulimits: []*units.Ulimit{},
						},
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_18",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						Resources: docker.Resources{
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
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						UsernsMode: container.UsernsMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_30",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						UsernsMode: container.UsernsMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_20",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						UTSMode: container.UTSMode("private"),
					},
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:5_20",
			container: docker.ContainerJSON{
				ContainerJSONBase: &docker.ContainerJSONBase{
					HostConfig: &docker.HostConfig{
						UTSMode: container.UTSMode("host"),
					},
				},
			},
			status: framework.FailStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:4_1",
			container: docker.ContainerJSON{
				Config: &docker.Config{
					User: "stackrox",
				},
			},
			status: framework.PassStatus,
		},
		{
			name: "CIS_Docker_v1_1_0:4_1",
			container: docker.ContainerJSON{
				Config: &docker.Config{
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
					c.container.State = &docker.ContainerState{
						Running: true,
					}
				} else {
					c.container.State.Running = true
				}
			} else {
				c.container.ContainerJSONBase = &docker.ContainerJSONBase{
					State: &docker.ContainerState{
						Running: true,
					},
				}
			}
			if c.container.HostConfig == nil {
				c.container.HostConfig = &docker.HostConfig{}
			}

			jsonData, err := json.Marshal(&docker.Data{
				Containers: []docker.ContainerJSON{
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

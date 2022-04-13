package docker

import (
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/docker/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerRuntimeChecks(t *testing.T) {
	cases := []struct {
		name      string
		container types.ContainerJSON
		status    storage.ComplianceState
	}{
		{
			name: "CIS_Docker_v1_2_0:5_25",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{"hello"},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_25",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{"hello", "no-new-privileges"},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_1",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					AppArmorProfile: "default",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_1",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					AppArmorProfile: "unconfined",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_13",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{}},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_13",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0"}},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name:   "CIS_Docker_v1_2_0:5_29",
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_29",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					Networks: map[string]struct{}{
						"bridge": {},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_3",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						CapAdd: []string{"CAP_SYS_ADMIN"},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_3",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						CapAdd: []string{},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_24",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CgroupParent: "docker",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_11",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CPUShares: 0,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_11",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							CPUShares: 10,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: nil,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: &types.Health{
							Status: "",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_26",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					State: &types.ContainerState{
						Health: &types.Health{
							Status: "yay",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_17",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Devices: []container.DeviceMapping{},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_17",
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
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_16",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						IpcMode: container.IpcMode("private"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_16",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						IpcMode: container.IpcMode("host"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_10",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Memory: 0,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_10",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Memory: 100,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_19",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Propagation: mount.PropagationShared,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_19",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Propagation: mount.PropagationPrivate,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_8",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{}},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_31",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/var/run/docker.sock",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_31",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/etc/passwd",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_15",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						PidMode: container.PidMode("private"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_15",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						PidMode: container.PidMode("host"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_28",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							PidsLimit: 0,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_28",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							PidsLimit: 10,
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_4",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Privileged: false,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_4",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Privileged: true,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_7",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostPort: "1025"}},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_7",
			container: types.ContainerJSON{
				NetworkSettings: &types.NetworkSettings{
					NetworkSettingsBase: types.NetworkSettingsBase{
						Ports: nat.PortMap{
							"80/tcp": []nat.PortBinding{{HostPort: "80"}},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_12",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						ReadonlyRootfs: true,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_12",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						ReadonlyRootfs: false,
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_14",
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
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_14",
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
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_21",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"seccomp:unconfined",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_21",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"seccomp:default",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_2",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_2",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						SecurityOpt: []string{
							"selinux",
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_5",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/etc",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_5",
			container: types.ContainerJSON{
				Mounts: []types.MountPoint{
					{
						Source: "/opt",
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_9",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						NetworkMode: container.NetworkMode("host"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_9",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						NetworkMode: container.NetworkMode("private"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_18",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						Resources: types.Resources{
							Ulimits: []*units.Ulimit{},
						},
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_18",
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
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_30",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UsernsMode: container.UsernsMode("private"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_30",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UsernsMode: container.UsernsMode("host"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:5_20",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UTSMode: container.UTSMode("private"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:5_20",
			container: types.ContainerJSON{
				ContainerJSONBase: &types.ContainerJSONBase{
					HostConfig: &types.HostConfig{
						UTSMode: container.UTSMode("host"),
					},
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "stackrox",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "root",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "0",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "0:0",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "0:70",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "70:70",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "root:root",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
		},
		{
			name: "CIS_Docker_v1_2_0:4_1",
			container: types.ContainerJSON{
				Config: &types.Config{
					User: "stackrox:stackrox",
				},
			},
			status: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(strings.Replace(c.name, ":", "-", -1), func(t *testing.T) {
			t.Parallel()

			standard := standards.NodeChecks[standards.CISDocker]
			require.NotNil(t, standard)
			check := standard[c.name]
			require.NotNil(t, check)

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
			mockNodeData := &standards.ComplianceData{
				DockerData: &types.Data{
					Containers: []types.ContainerJSON{
						c.container,
					},
				},
			}

			checkResults := check.CheckFunc(mockNodeData)
			require.Len(t, checkResults, 1)
			assert.Equal(t, c.status, checkResults[0].State)
		})
	}
}

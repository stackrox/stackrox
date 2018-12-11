package services

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

type mockDockerClient struct {
	tasks []swarm.Task
}

func (c mockDockerClient) TaskList(ctx context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	return c.tasks, nil
}

func (c mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	panic("not implemented")
}

func TestAsDeployment(t *testing.T) {
	t.Parallel()

	time100 := time.Unix(100, 0)

	cases := []struct {
		service  swarm.Service
		tasks    []swarm.Task
		expected *storage.Deployment
	}{
		{
			service: swarm.Service{
				ID: "fooID",
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: 100,
					},
				},
				Endpoint: swarm.Endpoint{
					Ports: []swarm.PortConfig{
						{
							Name:          "api",
							TargetPort:    80,
							PublishedPort: 8080,
							Protocol:      swarm.PortConfigProtocolTCP,
						},
					},
				},
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "foo",
						Labels: map[string]string{
							"key":      "value",
							"question": "answer",
						},
					},
					Mode: swarm.ServiceMode{
						Replicated: &swarm.ReplicatedService{
							Replicas: &[]uint64{10}[0],
						},
					},
					TaskTemplate: swarm.TaskSpec{
						ContainerSpec: &swarm.ContainerSpec{
							Args:    []string{"--flags", "--args"},
							Command: []string{"init"},
							Dir:     "/bin",
							Env:     []string{"LOGLEVEL=Warn", "JVMFLAGS=-Xms256m", "invalidENV"},
							Image:   "nginx:latest",
							User:    "root",
							Privileges: &swarm.Privileges{
								SELinuxContext: &swarm.SELinuxContext{
									User:  "user",
									Role:  "role",
									Type:  "type",
									Level: "level",
								},
							},
							Mounts: []mount.Mount{
								{
									Source:   "volumeSource",
									Type:     mount.TypeVolume,
									ReadOnly: true,
									Target:   "/var/data",
								},
							},
							Secrets: []*swarm.SecretReference{
								{
									File: &swarm.SecretReferenceFileTarget{
										Name: "path",
									},
									SecretID:   "id",
									SecretName: "name",
								},
							},
						},
						Resources: &swarm.ResourceRequirements{
							Reservations: &swarm.Resources{
								NanoCPUs:    1 * nanoCPUS,
								MemoryBytes: 1 * 1024 * 1024,
							},
							Limits: &swarm.Resources{
								NanoCPUs:    2 * nanoCPUS,
								MemoryBytes: 2 * 1024 * 1024,
							},
						},
					},
				},
				UpdateStatus: &swarm.UpdateStatus{
					CompletedAt: &time100,
				},
			},
			tasks: []swarm.Task{
				{
					NodeID: "mynode",
					Status: swarm.TaskStatus{
						ContainerStatus: swarm.ContainerStatus{
							ContainerID: "35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
						},
					},
				},
			},
			expected: &storage.Deployment{
				Id:        "fooID",
				Name:      "foo",
				Version:   "100",
				Namespace: defaultNamespace,
				Type:      "Replicated",
				Replicas:  10,
				Labels: map[string]string{
					"key":      "value",
					"question": "answer",
				},
				UpdatedAt: &timestamp.Timestamp{Seconds: 100},
				Containers: []*storage.Container{
					{
						Id: "c_fooID",
						Config: &storage.ContainerConfig{
							Args: []string{"--flags", "--args"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:   "LOGLEVEL",
									Value: "Warn",
								},
								{
									Key:   "JVMFLAGS",
									Value: "-Xms256m",
								},
							},
							Command:   []string{"init"},
							Directory: "/bin",
							User:      "root",
						},
						Image: &storage.Image{
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "library/nginx",
								Tag:      "latest",
								FullName: "docker.io/library/nginx:latest",
							},
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "name",
								Path: "/run/secrets/path",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Selinux: &storage.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
						Ports: []*storage.PortConfig{
							{
								Name:          "api",
								ContainerPort: 80,
								Protocol:      "tcp",
								Exposure:      storage.PortConfig_EXTERNAL,
								ExposedPort:   8080,
							},
						},
						Volumes: []*storage.Volume{
							{
								Name:        "volumeSource",
								Type:        "volume",
								ReadOnly:    true,
								Source:      "volumeSource",
								Destination: "/var/data",
							},
						},
						Resources: &storage.Resources{
							CpuCoresRequest: 1,
							CpuCoresLimit:   2,
							MemoryMbRequest: 1.00,
							MemoryMbLimit:   2.00,
						},
						Instances: []*storage.ContainerInstance{
							{
								InstanceId: &storage.ContainerInstanceID{
									ContainerRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
									Id:               "35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d7",
									Node:             "mynode",
								},
							},
						},
					},
				},
			},
		},
		{
			service: swarm.Service{
				ID: "fooID",
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: 100,
					},
				},
				Endpoint: swarm.Endpoint{
					Ports: []swarm.PortConfig{
						{
							Name:          "api",
							TargetPort:    80,
							PublishedPort: 8080,
							Protocol:      swarm.PortConfigProtocolTCP,
						},
					},
				},
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "foo",
						Labels: map[string]string{
							"key":      "value",
							"question": "answer",
						},
					},

					Mode: swarm.ServiceMode{
						Replicated: &swarm.ReplicatedService{
							Replicas: &[]uint64{10}[0],
						},
					},
					TaskTemplate: swarm.TaskSpec{
						ContainerSpec: &swarm.ContainerSpec{
							Args:    []string{"--flags", "--args"},
							Command: []string{"init"},
							Dir:     "/bin",
							Env:     []string{"LOGLEVEL=Warn", "JVMFLAGS=-Xms256m", "invalidENV"},
							Image:   "nginx:latest",
							User:    "root",
							Privileges: &swarm.Privileges{
								SELinuxContext: &swarm.SELinuxContext{
									User:  "user",
									Role:  "role",
									Type:  "type",
									Level: "level",
								},
							},
							Mounts: []mount.Mount{
								{
									Source:   "volumeSource",
									Type:     mount.TypeVolume,
									ReadOnly: true,
									Target:   "/var/data",
								},
							},
							Secrets: []*swarm.SecretReference{
								{
									File: &swarm.SecretReferenceFileTarget{
										Name: "path",
									},
									SecretID:   "id",
									SecretName: "name",
								},
							},
						},
						Resources: &swarm.ResourceRequirements{
							Reservations: &swarm.Resources{
								NanoCPUs:    1 * nanoCPUS,
								MemoryBytes: 1 * 1024 * 1024,
							},
							Limits: &swarm.Resources{
								NanoCPUs:    2 * nanoCPUS,
								MemoryBytes: 2 * 1024 * 1024,
							},
						},
					},
				},
				UpdateStatus: &swarm.UpdateStatus{
					CompletedAt: &time100,
				},
			},
			tasks: []swarm.Task{
				{
					NodeID: "mynode2",
					Status: swarm.TaskStatus{
						ContainerStatus: swarm.ContainerStatus{
							ContainerID: "35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d8",
						},
					},
				},
			},
			expected: &storage.Deployment{
				Id:        "fooID",
				Name:      "foo",
				Version:   "100",
				Namespace: defaultNamespace,
				Type:      "Replicated",
				Replicas:  10,
				Labels: map[string]string{
					"key":      "value",
					"question": "answer",
				},
				UpdatedAt: &timestamp.Timestamp{Seconds: 100},
				Containers: []*storage.Container{
					{
						Id: "c_fooID",
						Config: &storage.ContainerConfig{
							Args: []string{"--flags", "--args"},
							Env: []*storage.ContainerConfig_EnvironmentConfig{
								{
									Key:   "LOGLEVEL",
									Value: "Warn",
								},
								{
									Key:   "JVMFLAGS",
									Value: "-Xms256m",
								},
							},
							Command:   []string{"init"},
							Directory: "/bin",
							User:      "root",
						},
						Image: &storage.Image{
							Name: &storage.ImageName{
								Registry: "docker.io",
								Remote:   "library/nginx",
								Tag:      "latest",
								FullName: "docker.io/library/nginx:latest",
							},
						},
						Secrets: []*storage.EmbeddedSecret{
							{
								Name: "name",
								Path: "/run/secrets/path",
							},
						},
						SecurityContext: &storage.SecurityContext{
							Selinux: &storage.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
						Ports: []*storage.PortConfig{
							{
								Name:          "api",
								ContainerPort: 80,
								Protocol:      "tcp",
								Exposure:      storage.PortConfig_EXTERNAL,
								ExposedPort:   8080,
							},
						},
						Volumes: []*storage.Volume{
							{
								Name:        "volumeSource",
								Type:        "volume",
								ReadOnly:    true,
								Source:      "volumeSource",
								Destination: "/var/data",
							},
						},
						Resources: &storage.Resources{
							CpuCoresRequest: 1,
							CpuCoresLimit:   2,
							MemoryMbRequest: 1.00,
							MemoryMbLimit:   2.00,
						},
						Instances: []*storage.ContainerInstance{
							{
								InstanceId: &storage.ContainerInstanceID{
									ContainerRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
									Id:               "35669191c32a9cfb532e5d79b09f2b0926c0faf27e7543f1fbe433bd94ae78d8",
									Node:             "mynode2",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		cli := mockDockerClient{
			tasks: c.tasks,
		}
		got := serviceWrap(c.service).asDeployment(cli, false)

		assert.Equal(t, c.expected, got)
	}
}

func TestConvertNanoCPUsToCores(t *testing.T) {
	cases := []struct {
		expected float32
		value    int64
	}{
		{
			expected: 1,
			value:    nanoCPUS,
		},
		{
			expected: 0,
			value:    0,
		},
		{
			expected: 1000,
			value:    1000000000000,
		},
		{
			expected: 1.1,
			value:    nanoCPUS + nanoCPUS/10,
		},
	}
	for _, c := range cases {
		t.Run(strconv.FormatFloat(float64(c.expected), 'e', -1, 32), func(t *testing.T) {
			assert.InDelta(t, c.expected, convertNanoCPUsToCores(c.value), 0.01)
		})
	}
}

func TestConvertMemoryBytesToMb(t *testing.T) {
	cases := []struct {
		expected float32
		value    int64
	}{
		{
			expected: 1,
			value:    megabyte,
		},
		{
			expected: 0,
			value:    0,
		},
		{
			expected: 1024,
			value:    1024 * 1024 * 1024,
		},
		{
			expected: 1.1,
			value:    megabyte + megabyte/10,
		},
	}
	for _, c := range cases {
		t.Run(strconv.FormatFloat(float64(c.expected), 'e', -1, 32), func(t *testing.T) {
			assert.InDelta(t, c.expected, convertMemoryBytesToMb(c.value), 0.01)
		})
	}
}

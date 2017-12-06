package listener

import (
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestAsDeployment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		service  swarm.Service
		expected *v1.Deployment
	}{
		{
			service: swarm.Service{
				ID: "fooID",
				Meta: swarm.Meta{
					Version: swarm.Version{
						Index: 100,
					},
				},
				Spec: swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "foo",
					},
					Mode: swarm.ServiceMode{
						Replicated: &swarm.ReplicatedService{
							Replicas: &[]uint64{1}[0],
						},
					},
					TaskTemplate: swarm.TaskSpec{
						ContainerSpec: swarm.ContainerSpec{
							Image: "nginx:latest",
						},
					},
				},
				UpdateStatus: swarm.UpdateStatus{
					CompletedAt: time.Unix(100, 0),
				},
			},
			expected: &v1.Deployment{
				Id:        "fooID",
				Name:      "foo",
				Version:   "100",
				Type:      "Replicated",
				UpdatedAt: &timestamp.Timestamp{Seconds: 100},
				Image: &v1.Image{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
				},
			},
		},
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		got := serviceWrap(c.service).asDeployment(cli)

		assert.Equal(t, c.expected, got)
	}

}

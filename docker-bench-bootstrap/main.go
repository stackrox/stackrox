package main

import (
	"context"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const (
	dockerBenchImageEnv = "ROX_DOCKER_BENCH_IMAGE"
)

func main() {
	image := "stackrox/docker-bench:latest"
	if imageEnv := os.Getenv(dockerBenchImageEnv); imageEnv != "" {
		image = imageEnv
	}

	client, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Unable to connect to docker client: %+v", err)
	}

	strVolumes := []string{
		"/var/run/docker.sock:/var/run/docker.sock",
		"/var/lib:/var/lib",
		"/etc:/etc",
		"/var/log/audit:/var/log/audit",
		"/lib/systemd:/lib/systemd",
		"/usr/lib/systemd:/usr/lib/systemd",
	}

	volumeMap := make(map[string]struct{})
	for _, vol := range strVolumes {
		volumeMap[vol] = struct{}{}
	}

	containerConfig := &container.Config{
		Env:     os.Environ(),
		Image:   image,
		Volumes: volumeMap,
	}
	hostConfig := &container.HostConfig{
		Binds:   strVolumes,
		PidMode: container.PidMode("host"),
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"apollo_net": {},
		},
	}

	body, err := client.ContainerCreate(context.Background(), containerConfig, hostConfig, networkingConfig, "")
	if err != nil {
		log.Fatalf("Error creating docker-bench container: %+v", err)
	}

	if err := client.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("Error starting docker-bench container: %+v", err)
	}

	if _, err := client.ContainerWait(context.Background(), body.ID); err != nil {
		log.Fatalf("error waiting for container %v to finish", body.ID)
	}
}

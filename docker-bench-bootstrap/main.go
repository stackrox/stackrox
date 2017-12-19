package main

import (
	"context"
	"log"
	"os"

	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

const (
	dockerBenchImageEnv = "ROX_DOCKER_BENCH_IMAGE"
)

func main() {
	image := "stackrox/apollo:latest"
	if imageEnv := os.Getenv(dockerBenchImageEnv); imageEnv != "" {
		image = imageEnv
	}

	client, err := docker.NewClient()
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
		Env:        os.Environ(),
		Image:      image,
		Volumes:    volumeMap,
		Entrypoint: []string{"docker-bench"},
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

	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	body, err := client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, "")
	if err != nil {
		log.Fatalf("Error creating docker-bench container: %+v", err)
	}
	ctx, cancel = docker.TimeoutContext()
	defer cancel()
	if err := client.ContainerStart(ctx, body.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("Error starting docker-bench container: %+v", err)
	}

	okC, errC := client.ContainerWait(context.Background(), body.ID, container.WaitConditionNotRunning)
	select {
	case <-okC:
		return
	case err := <-errC:
		log.Fatalf("error waiting for container %v to finish: %s", body.ID, err)
	}
}

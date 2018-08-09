package main

import (
	"context"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/env"
)

func main() {
	image := env.Image.Setting()

	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Unable to connect to docker client: %+v", err)
	}
	volumeMap := make(map[string]struct{})
	for _, vol := range benchmarks.BenchmarkMounts {
		volumeMap[vol] = struct{}{}
	}
	containerConfig := &container.Config{
		Env:        os.Environ(),
		Image:      image,
		Volumes:    volumeMap,
		Entrypoint: []string{benchmarks.BenchmarkCommand},
	}
	hostConfig := &container.HostConfig{
		Binds:      benchmarks.BenchmarkMounts,
		PidMode:    container.PidMode("host"),
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"prevent_net": {},
		},
	}

	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	body, err := client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, "")
	if err != nil {
		log.Fatalf("Error creating benchmarks container: %+v", err)
	}

	ctx, cancel = docker.TimeoutContext()
	defer cancel()
	if err := client.ContainerStart(ctx, body.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("Error starting benchmarks container: %+v", err)
	}

	okC, errC := client.ContainerWait(context.Background(), body.ID, container.WaitConditionNotRunning)
	select {
	case <-okC:
		return
	case err := <-errC:
		log.Fatalf("error waiting for container %v to finish: %s", body.ID, err)
	}
}

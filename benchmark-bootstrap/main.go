package main

import (
	"context"
	"log"
	"os"

	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

func main() {
	image := env.Image.Setting()

	client, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Unable to connect to docker client: %+v", err)
	}

	strVolumes := []string{
		"/var/run/docker.sock:/var/run/docker.sock",
		"/var/lib:/var/lib:ro",
		"/etc:/etc:ro",
		"/var/log/audit:/var/log/audit:ro",
		"/lib/systemd:/lib/systemd:ro",
		"/usr/lib/systemd:/usr/lib/systemd:ro",
	}

	volumeMap := make(map[string]struct{})
	for _, vol := range strVolumes {
		volumeMap[vol] = struct{}{}
	}

	containerConfig := &container.Config{
		Env:        os.Environ(),
		Image:      image,
		Volumes:    volumeMap,
		Entrypoint: []string{"benchmarks"},
	}
	hostConfig := &container.HostConfig{
		Binds:      strVolumes,
		PidMode:    container.PidMode("host"),
		AutoRemove: true,
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

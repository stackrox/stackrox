package docker

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
)

const timeout = 30 * time.Second

var (
	pathsForDockerSocket = []string{
		"unix:///host/run/docker.sock",
		"unix:///host/var/run/docker.sock",
	}

	log = logging.LoggerForModule()
)

// Data is the wrapper around all of the Docker info required for compliance
type Data struct {
	Info       types.Info
	Containers []types.ContainerJSON
	Images     []ImageWrap
}

func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ImageWrap is a wrapper around a docker image because normally the image doesn't give the history
type ImageWrap struct {
	Image   types.ImageInspect          `json:"image"`
	History []image.HistoryResponseItem `json:"history"`
}

func getClient() (client *client.Client, err error) {
	for _, p := range pathsForDockerSocket {
		os.Setenv("DOCKER_HOST", p)
		client, err = docker.NewClient()
		if err == nil {
			return
		}
	}
	return
}

// GetDockerData returns the marshalled JSON from scraping Docker
func GetDockerData() (*compliance.JSONDataChunk, error) {
	var dockerData Data

	client, err := getClient()
	if err != nil {
		return nil, err
	}

	dockerData.Info, err = getInfo(client)
	if err != nil {
		return nil, err
	}

	dockerData.Containers, err = getContainers(client)
	if err != nil {
		return nil, err
	}

	dockerData.Images, err = getImages(client)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(&dockerData)
	if err != nil {
		return nil, err
	}

	return &compliance.JSONDataChunk{
		Json: bytes,
	}, nil
}

func getInfo(c *client.Client) (types.Info, error) {
	ctx, cancel := getContext()
	defer cancel()

	return c.Info(ctx)
}

func inspectContainer(client *client.Client, id string) (types.ContainerJSON, error) {
	ctx, cancel := getContext()
	defer cancel()
	return client.ContainerInspect(ctx, id)
}

func getContainers(c *client.Client) ([]types.ContainerJSON, error) {
	ctx, cancel := getContext()
	defer cancel()

	containerList, err := c.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	containers := make([]types.ContainerJSON, 0, len(containerList))
	for _, container := range containerList {
		containerJSON, err := inspectContainer(c, container.ID)
		if client.IsErrContainerNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		containers = append(containers, containerJSON)
	}
	return containers, nil
}

func getImageHistory(c *client.Client, id string) ([]image.HistoryResponseItem, error) {
	ctx, cancel := getContext()
	defer cancel()
	return c.ImageHistory(ctx, id)
}

func inspectImage(c *client.Client, id string) (types.ImageInspect, error) {
	ctx, cancel := getContext()
	defer cancel()
	i, _, err := c.ImageInspectWithRaw(ctx, id)
	return i, err
}

func getImages(c *client.Client) ([]ImageWrap, error) {
	ctx, cancel := getContext()
	defer cancel()

	// Saying All with images gives you a bunch of worthless layers
	imageList, err := c.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return nil, err
	}

	images := make([]ImageWrap, 0, len(imageList))
	for _, i := range imageList {
		image, err := inspectImage(c, i.ID)
		if client.IsErrImageNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		histories, err := getImageHistory(c, i.ID)
		if client.IsErrImageNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		images = append(images, ImageWrap{Image: image, History: histories})
	}
	return images, nil
}

package docker

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/time/rate"
)

const timeout = 30 * time.Second

var (
	pathsForDockerSocket = []string{
		"unix:///host/run/docker.sock",
		"unix:///host/var/run/docker.sock",
	}

	log = logging.LoggerForModule()

	dockerRateLimiter = rate.NewLimiter(rate.Every(50*time.Millisecond), 1)
)

// Data is the wrapper around all of the Docker info required for compliance
type Data struct {
	Info          types.Info
	Containers    []types.ContainerJSON
	Images        []ImageWrap
	BridgeNetwork types.NetworkResource
}

func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ImageWrap is a wrapper around a docker image because normally the image doesn't give the history
type ImageWrap struct {
	Image   types.ImageInspect          `json:"image"`
	History []image.HistoryResponseItem `json:"history"`
}

// Config returns an empty config if one does not exist or the config from the Image object
func (i ImageWrap) Config() *container.Config {
	if i.Image.Config == nil {
		return &container.Config{}
	}
	return i.Image.Config
}

// Name attempts to return a human-readable registry-based name, but will fall back to ID if it cannot
func (i ImageWrap) Name() string {
	if len(i.Image.RepoTags) != 0 {
		return i.Image.RepoTags[0]
	}
	if len(i.Image.RepoDigests) != 0 {
		return i.Image.RepoDigests[0]
	}
	return i.Image.ID
}

func checkClient(c *client.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := c.Info(ctx)
	return err
}

func getClient() (*client.Client, error) {
	errorList := errorhelpers.NewErrorList("Docker client")
	for _, p := range pathsForDockerSocket {
		log.Infof("Trying to create client with: %s", p)
		client, err := docker.NewClientWithPath(p)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if err := checkClient(client); err != nil {
			errorList.AddError(err)
			continue
		}
		return client, nil
	}
	return nil, errorList.ToError()
}

// GetDockerData returns the marshaled JSON from scraping Docker
func GetDockerData() (*compliance.GZIPDataChunk, error) {
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

	dockerData.BridgeNetwork, err = getBridgeNetwork(client)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz, err = gzip.NewWriterLevel(gz, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if err := json.NewEncoder(gz).Encode(&dockerData); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return &compliance.GZIPDataChunk{
		Gzip: buf.Bytes(),
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

	dockerRateLimiter.Wait(context.Background())
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
		dockerRateLimiter.Wait(context.Background())
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

func getBridgeNetwork(c *client.Client) (types.NetworkResource, error) {
	listFilters := filters.NewArgs()
	listFilters.Add("Name", "bridge")
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	return c.NetworkInspect(ctx, "bridge", types.NetworkInspectOptions{})
}

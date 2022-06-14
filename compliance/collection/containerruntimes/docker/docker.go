package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/pkg/docker"
	"github.com/stackrox/stackrox/pkg/docker/client"
	internalTypes "github.com/stackrox/stackrox/pkg/docker/types"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/logging"
	"golang.org/x/time/rate"
)

const (
	timeout = 30 * time.Second

	logInterval = 5 * time.Second
)

var (
	pathsForDockerSocket = []string{
		"unix:///host/run/docker.sock",
		"unix:///host/var/run/docker.sock",
	}

	log = logging.LoggerForModule()

	dockerRateLimiter = rate.NewLimiter(rate.Every(10*time.Millisecond), 1)

	// filter out all the containers with these labels
	whiteListContainersWithLabels = map[string]string{
		"com.stackrox.io/service": "compliance", // Ref: sensor/common/compliance/command_handler_impl.go:189
	}
)

func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
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
		log.Infof("Trying to create client with path %q", p)
		client, err := docker.NewClientWithPath(p)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		if err := checkClient(client); err != nil {
			errorList.AddError(err)
			continue
		}
		log.Infof("Successfully created docker client with path %q", p)
		return client, nil
	}
	return nil, errorList.ToError()
}

// GetDockerData returns the marshaled JSON from scraping Docker
func GetDockerData() (*internalTypes.Data, *compliance.ContainerRuntimeInfo, error) {
	var dockerData internalTypes.Data

	client, err := getClient()
	if err != nil {
		return nil, nil, err
	}

	dockerData.Info, err = getInfo(client)
	if err != nil {
		return nil, nil, err
	}

	dockerData.Containers, err = getContainers(client)
	if err != nil {
		return nil, nil, err
	}

	dockerData.Images, err = getImages(client)
	if err != nil {
		return nil, nil, err
	}

	dockerData.BridgeNetwork, err = getBridgeNetwork(client)
	if err != nil {
		return nil, nil, err
	}

	return &dockerData, toStandardizedInfo(&dockerData), nil
}

func getInfo(c *client.Client) (types.Info, error) {
	ctx, cancel := getContext()
	defer cancel()

	return c.Info(ctx)
}

func inspectContainer(client *client.Client, id string) (*internalTypes.ContainerJSON, error) {
	ctx, cancel := getContext()
	defer cancel()

	return client.ContainerInspect(ctx, id, false)
}

func containerMatchesWhitelist(container *internalTypes.ContainerList) bool {
	for k, v := range whiteListContainersWithLabels {
		if container.Labels[k] == v {
			return true
		}
	}
	return false
}

func getContainers(c *client.Client) ([]internalTypes.ContainerJSON, error) {
	ctx, cancel := getContext()
	defer cancel()

	containerList, err := c.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	log.Infof("Listed %d containers", len(containerList))

	lastLogTS := time.Now()
	containers := make([]internalTypes.ContainerJSON, 0, len(containerList))
	for i, container := range containerList {
		if err := dockerRateLimiter.Wait(context.Background()); err != nil {
			return nil, err
		}
		if time.Since(lastLogTS) >= logInterval {
			log.Infof("Processed %d/%d containers", i, len(containerList))
			lastLogTS = time.Now()
		}
		if containerMatchesWhitelist(container) {
			continue
		}
		containerJSON, err := inspectContainer(c, container.ID)
		if client.IsErrContainerNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		containers = append(containers, *containerJSON)
	}
	log.Info("Successfully listed all containers")
	return containers, nil
}

func getImageHistory(c *client.Client, id string) ([]image.HistoryResponseItem, error) {
	ctx, cancel := getContext()
	defer cancel()
	return c.ImageHistory(ctx, id)
}

func inspectImage(c *client.Client, id string) (*internalTypes.ImageInspect, error) {
	ctx, cancel := getContext()
	defer cancel()
	return c.ImageInspect(ctx, id)
}

func getImages(c *client.Client) ([]internalTypes.ImageWrap, error) {
	ctx, cancel := getContext()
	defer cancel()

	// Saying All with images gives you a bunch of worthless layers
	imageList, err := c.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return nil, err
	}

	log.Infof("Listed %d images", len(imageList))
	lastLogTS := time.Now()

	images := make([]internalTypes.ImageWrap, 0, len(imageList))
	for i, img := range imageList {
		if time.Since(lastLogTS) >= logInterval {
			log.Infof("Processed %d/%d images", i, len(imageList))
			lastLogTS = time.Now()
		}
		if err := dockerRateLimiter.Wait(context.Background()); err != nil {
			return nil, err
		}
		image, err := inspectImage(c, img.ID)
		if client.IsErrImageNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		histories, err := getImageHistory(c, img.ID)
		if client.IsErrImageNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		images = append(images, internalTypes.ImageWrap{Image: *image, History: histories})
	}
	log.Info("Successfully collected all images")
	return images, nil
}

func getBridgeNetwork(c *client.Client) (types.NetworkResource, error) {
	ctx, cancel := docker.TimeoutContext()
	defer cancel()

	network, err := c.NetworkInspect(ctx, "bridge", types.NetworkInspectOptions{})
	if client.IsErrNotFound(err) {
		return types.NetworkResource{}, nil
	}
	return network, err
}

package dockerhub

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/registries"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/heroku/docker-registry-client/registry"
)

type dockerHubClient struct {
	hub *registry.Registry
}

// Parse out the layer JSON
type v1Compatibility struct {
	Created time.Time `json:"created"`
	Author  string    `json:"author"`
}

func new(config map[string]string) (registryTypes.ImageRegistry, error) {
	url, ok := config["url"]
	if !ok {
		return nil, errors.New("Config parameters 'url' must be defined for dockerhub plugin")
	}
	username, ok := config["username"]
	if !ok {
		return nil, errors.New("Config parameters 'username' must be defined for dockerhub plugin")
	}
	password, ok := config["password"]
	if !ok {
		return nil, errors.New("Config parameters 'password' must be defined for dockerhub plugin")
	}

	hub, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}
	return &dockerHubClient{
		hub: hub,
	}, nil
}

// Metadata returns the metadata via this registries implementation
func (d *dockerHubClient) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	imageName := fmt.Sprintf("%v/%v", image.Registry, image.Repo)
	manifest, err := d.hub.Manifest(imageName, image.Tag)
	if err != nil {
		return nil, err
	}

	// Get the latest layer and author
	var latest v1Compatibility
	for _, layer := range manifest.History {
		var compat v1Compatibility
		if err := json.Unmarshal([]byte(layer.V1Compatibility), &compat); err != nil {
			return nil, err
		}
		if latest.Created.After(latest.Created) {
			latest = compat
		}
	}
	imageMetadata := &v1.ImageMetadata{
		//Digest:  convertedDigest,
		Created: latest.Created.UnixNano(),
		Author:  latest.Author,
	}
	return imageMetadata, nil
}

// Test tests the current registry and makes sure that it is working properly
func (d *dockerHubClient) Test() error {
	return nil
}

func init() {
	registries.Registry["dockerhub"] = new
}

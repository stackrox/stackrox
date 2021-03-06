package types

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
)

// Data is the wrapper around all of the Docker info required for compliance
// easyjson:json
type Data struct {
	Info          types.Info
	Containers    []ContainerJSON
	Images        []ImageWrap
	BridgeNetwork types.NetworkResource
}

// ImageWrap is a wrapper around a docker image because normally the image doesn't give the history
type ImageWrap struct {
	Image   ImageInspect                `json:"image"`
	History []image.HistoryResponseItem `json:"history"`
}

// Config returns an empty config if one does not exist or the config from the Image object
func (i ImageWrap) Config() *Config {
	if i.Image.Config == nil {
		return &Config{}
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

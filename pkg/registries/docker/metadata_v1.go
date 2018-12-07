package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"github.com/gogo/protobuf/types"
	"github.com/opencontainers/go-digest"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	instructionTypes "github.com/stackrox/rox/pkg/registries/types"
)

var scrubPrefixes = []string{
	"/bin/sh -c #(nop)",
	"/bin/sh -c ",
}

func lineToInstructionAndValue(line string) (instruction string, value string) {
	line = strings.TrimSpace(line)
	for _, prefixToRemove := range scrubPrefixes {
		line = strings.TrimPrefix(line, prefixToRemove)
	}
	line = strings.TrimSpace(line)
	for instructionType := range instructionTypes.DockerfileInstructionSet {
		if strings.HasPrefix(line, instructionType) {
			instruction = instructionType
			value = strings.TrimPrefix(line, instruction+" ")
			return
		}
	}
	instruction = "RUN"
	value = line
	return
}

func convertImageToDockerFileLine(img *image.V1Image) *storage.ImageLayer {
	line := strings.Join(img.ContainerConfig.Cmd, " ")
	line = strings.Join(strings.Fields(line), " ")
	instruction, value := lineToInstructionAndValue(line)
	protoTS, err := types.TimestampProto(img.Created)
	if err != nil {
		log.Error(err)
	}
	return &storage.ImageLayer{
		Instruction: instruction,
		Value:       value,
		Created:     protoTS,
		Author:      img.Author,
	}
}

func (r *Registry) populateV1DataFromManifest(manifest *schema1.SignedManifest, ref string) (*storage.ImageMetadata, error) {
	// Get the latest layer and author
	var latest storage.ImageLayer
	var layers []*storage.ImageLayer
	for i := len(manifest.History) - 1; i > -1; i-- {
		historyLayer := manifest.History[i]
		var v1Image image.V1Image
		if err := json.Unmarshal([]byte(historyLayer.V1Compatibility), &v1Image); err != nil {
			return nil, fmt.Errorf("Failed unmarshalling v1 capability: %s", err)
		}
		layer := convertImageToDockerFileLine(&v1Image)
		if protoconv.CompareProtoTimestamps(layer.Created, latest.Created) == 1 {
			latest = *layer
		}
		layers = append(layers, layer)
	}
	// Orient the layers to be oldest to newest
	fsLayers := make([]string, 0, len(manifest.FSLayers))
	for i := len(manifest.FSLayers) - 1; i > -1; i-- {
		fsLayers = append(fsLayers, manifest.FSLayers[i].BlobSum.String())
	}

	return &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Digest:  ref,
			Created: latest.Created,
			Author:  latest.Author,
			Layers:  layers,
		},
		LayerShas: fsLayers,
	}, nil
}

func (r *Registry) handleV1SignedManifest(remote, ref string) (*storage.ImageMetadata, error) {
	manifest, err := r.client.SignedManifest(remote, ref)
	if err != nil {
		return nil, err
	}
	return r.populateV1DataFromManifest(manifest, ref)
}

func (r *Registry) handleV1Manifest(remote, ref string) (*storage.ImageMetadata, error) {
	manifest, err := r.client.Manifest(remote, ref)
	if err != nil {
		return nil, err
	}
	return r.populateV1DataFromManifest(manifest, ref)
}

func (r *Registry) handleV1ManifestLayer(remote string, ref digest.Digest) (*storage.V1Metadata, error) {
	v1r, err := r.client.DownloadLayer(remote, ref)
	if err != nil {
		return nil, err
	}
	val, err := ioutil.ReadAll(v1r)
	if err != nil {
		return nil, err
	}
	img := &image.Image{}
	if err := json.Unmarshal(val, img); err != nil {
		return nil, err
	}

	var layers []*storage.ImageLayer
	for _, h := range img.History {
		// See github.com/moby/moby/image/image.go
		instruction, value := lineToInstructionAndValue(h.CreatedBy)
		layers = append(layers, &storage.ImageLayer{
			Created:     protoconv.ConvertTimeToTimestamp(h.Created),
			Author:      h.Author,
			Instruction: instruction,
			Value:       value,
			Empty:       h.EmptyLayer,
		})
	}
	var metadata = &storage.V1Metadata{}
	if len(layers) != 0 {
		lastLayer := layers[len(layers)-1]
		metadata.Author = lastLayer.Author
		metadata.Created = lastLayer.Created
	}
	metadata.Layers = layers

	return metadata, nil
}

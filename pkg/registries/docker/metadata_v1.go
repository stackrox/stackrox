package docker

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/gogo/protobuf/types"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
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

func convertImageToDockerFileLine(img *v1.Image) *storage.ImageLayer {
	line := strings.Join(img.Config.Cmd, " ")
	line = strings.Join(strings.Fields(line), " ")
	instruction, value := lineToInstructionAndValue(line)
	created := time.Time{}
	if img.Created != nil {
		created = *img.Created
	}
	protoTS, err := types.TimestampProto(created)
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
	labels := make(map[string]string)
	for i := len(manifest.History) - 1; i > -1; i-- {
		historyLayer := manifest.History[i]
		var v1Image v1.Image
		if err := json.Unmarshal([]byte(historyLayer.V1Compatibility), &v1Image); err != nil {
			return nil, errors.Wrap(err, "Failed unmarshalling v1 capability")
		}
		layer := convertImageToDockerFileLine(&v1Image)
		if layer.Created.Compare(latest.Created) == 1 {
			latest = *layer
		}
		layers = append(layers, layer)
		// Last label takes precedence and there seems to be a separate image object per layer
		for labelKey, labelValue := range v1Image.Config.Labels {
			labels[labelKey] = labelValue
		}
	}
	// Orient the layers to be oldest to newest
	fsLayers := make([]string, 0, len(manifest.FSLayers))
	for i := len(manifest.FSLayers) - 1; i > -1; i-- {
		fsLayers = append(fsLayers, manifest.FSLayers[i].BlobSum.String())
	}

	// Nil out empty label maps to be consistent with handleV1ManifestLayer()
	if len(labels) == 0 {
		labels = nil
	}

	return &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Digest:  ref,
			Created: latest.Created,
			Author:  latest.Author,
			Layers:  layers,
			Labels:  labels,
		},
		LayerShas: fsLayers,
	}, nil
}

// HandleV1SignedManifest takes in a signed v1 ref and returns the image metadata
func HandleV1SignedManifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	manifest, err := r.Client.SignedManifest(remote, ref)
	if err != nil {
		return nil, err
	}
	return r.populateV1DataFromManifest(manifest, ref)
}

// HandleV1Manifest takes in a v1 ref and returns the image metadata
func HandleV1Manifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	manifest, err := r.Client.Manifest(remote, ref)
	if err != nil {
		return nil, err
	}
	return r.populateV1DataFromManifest(manifest, ref)
}

func (r *Registry) handleV1ManifestLayer(remote string, ref digest.Digest) (*storage.V1Metadata, error) {
	v1r, err := r.Client.DownloadLayer(remote, ref)
	if err != nil {
		return nil, err
	}
	val, err := io.ReadAll(v1r)
	if err != nil {
		return nil, err
	}
	img := &v1.Image{}
	if err := json.Unmarshal(val, img); err != nil {
		return nil, err
	}

	var layers []*storage.ImageLayer
	for _, h := range img.History {
		instruction, value := lineToInstructionAndValue(h.CreatedBy)
		layers = append(layers, &storage.ImageLayer{
			Created:     protoconv.ConvertTimeToTimestampOrNow(h.Created),
			Author:      h.Author,
			Instruction: instruction,
			Value:       value,
			Empty:       h.EmptyLayer,
		})
	}

	var metadata = &storage.V1Metadata{
		Digest:  ref.String(),
		Created: protoconv.ConvertTimeToTimestampOrNow(img.Created),
		Labels:  img.Config.Labels,
	}

	metadata.Volumes = make([]string, 0, len(img.Config.Volumes))
	for k := range img.Config.Volumes {
		metadata.Volumes = append(metadata.Volumes, k)
	}

	metadata.User = "root"
	if img.Config.User != "" {
		metadata.User = img.Config.User
	}
	metadata.Command = img.Config.Cmd
	metadata.Entrypoint = img.Config.Entrypoint

	if len(layers) != 0 {
		lastLayer := layers[len(layers)-1]
		metadata.Author = lastLayer.Author
	}
	metadata.Layers = layers
	return metadata, nil
}

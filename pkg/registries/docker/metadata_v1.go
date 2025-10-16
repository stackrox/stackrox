package docker

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
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
	var created time.Time
	if img.Created != nil {
		created = *img.Created
	}
	protoTS, err := protocompat.ConvertTimeToTimestampOrError(created)
	if err != nil {
		log.Error(err)
	}
	il := &storage.ImageLayer{}
	il.SetInstruction(instruction)
	il.SetValue(value)
	il.SetCreated(protoTS)
	il.SetAuthor(img.Author)
	return il
}

func (r *Registry) populateV1DataFromManifest(manifest *schema1.SignedManifest, ref string) (*storage.ImageMetadata, error) {
	// Get the latest layer and author
	var latest *storage.ImageLayer
	var layers []*storage.ImageLayer
	labels := make(map[string]string)
	for i := len(manifest.History) - 1; i > -1; i-- {
		historyLayer := manifest.History[i]
		var v1Image v1.Image
		if err := json.Unmarshal([]byte(historyLayer.V1Compatibility), &v1Image); err != nil {
			return nil, errors.Wrap(err, "Failed unmarshalling v1 capability")
		}
		layer := convertImageToDockerFileLine(&v1Image)
		if protocompat.CompareTimestamps(layer.GetCreated(), latest.GetCreated()) == 1 {
			latest = layer
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

	v1m := &storage.V1Metadata{}
	v1m.SetDigest(ref)
	v1m.SetCreated(latest.GetCreated())
	v1m.SetAuthor(latest.GetAuthor())
	v1m.SetLayers(layers)
	v1m.SetLabels(labels)
	im := &storage.ImageMetadata{}
	im.SetV1(v1m)
	im.SetLayerShas(fsLayers)
	return im, nil
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
		il := &storage.ImageLayer{}
		il.SetCreated(protoconv.ConvertTimeToTimestampOrNow(h.Created))
		il.SetAuthor(h.Author)
		il.SetInstruction(instruction)
		il.SetValue(value)
		il.SetEmpty(h.EmptyLayer)
		layers = append(layers, il)
	}

	v1m := &storage.V1Metadata{}
	v1m.SetDigest(ref.String())
	v1m.SetCreated(protoconv.ConvertTimeToTimestampOrNow(img.Created))
	v1m.SetLabels(img.Config.Labels)
	var metadata = v1m

	metadata.SetVolumes(make([]string, 0, len(img.Config.Volumes)))
	for k := range img.Config.Volumes {
		metadata.SetVolumes(append(metadata.GetVolumes(), k))
	}

	metadata.SetUser("root")
	if img.Config.User != "" {
		metadata.SetUser(img.Config.User)
	}
	metadata.SetCommand(img.Config.Cmd)
	metadata.SetEntrypoint(img.Config.Entrypoint)

	if len(layers) != 0 {
		lastLayer := layers[len(layers)-1]
		metadata.SetAuthor(lastLayer.GetAuthor())
	}
	metadata.SetLayers(layers)
	return metadata, nil
}

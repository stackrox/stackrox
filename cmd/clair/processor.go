package main

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	"github.com/heroku/docker-registry-client/registry"
)

type scanProcessor struct {
	dockerRegistry *registry.Registry

	clair *clairClient
}

func newProcessor(url string, auth *basicAuth, clair *clairClient) (*scanProcessor, error) {
	fullURL, err := urlfmt.FormatURL(url, true, false)
	if err != nil {
		log.Fatalf("Could not parse registry endpoint %v: %v", url, err)
	}

	dockerRegistry, err := registry.New(fullURL, auth.username, auth.password)
	if err != nil {
		return nil, err
	}
	return &scanProcessor{
		clair:          clair,
		dockerRegistry: dockerRegistry,
	}, nil
}

func isEmptyLayer(blobSum string) bool {
	return blobSum == emptyLayerBlobSum || blobSum == legacyEmptyLayerBlobSum
}

func (p *scanProcessor) getHeaders() map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %v", p.dockerRegistry.Transport.GetToken()),
	}
}

func (p *scanProcessor) processImage(image *v1.Image) error {
	layers, err := p.fetchLayers(image)
	if err != nil {
		return err
	}
	if len(layers) == 0 {
		return fmt.Errorf("No layers to process for image %s", image.GetName().GetFullName())
	}
	headers := p.getHeaders()
	return p.clair.analyzeRemoteImage(p.dockerRegistry.URL, image, layers, headers)
}

func (p *scanProcessor) fetchLayers(image *v1.Image) ([]string, error) {
	var layers []string
	m2, err := p.dockerRegistry.ManifestV2(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil || len(m2.Layers) == 0 {
		// fall back to v1 if no v2
		m1, err := p.dockerRegistry.Manifest(image.GetName().GetRemote(), image.GetName().GetTag())
		if err != nil {
			return nil, err
		}
		// FSLayers has the most recent layer first, append them so that parent layers are first in the slice
		for i := len(m1.FSLayers) - 1; i >= 0; i-- {
			layer := m1.FSLayers[i]
			if isEmptyLayer(layer.BlobSum.String()) {
				continue
			}
			layers = append(layers, layer.BlobSum.String())
		}
		return layers, nil
	}
	for _, layer := range m2.Layers {
		if isEmptyLayer(layer.Digest.String()) {
			continue
		}
		layers = append(layers, layer.Digest.String())
	}
	return layers, nil
}

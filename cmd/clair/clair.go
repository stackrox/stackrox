package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	clairV1 "github.com/coreos/clair/api/v1"
)

const (
	emptyLayerBlobSum       = "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	legacyEmptyLayerBlobSum = "sha256:5f70bf18a086007016e948b04aed3b82103a36bea41755b6cddfaf10ace3c6ef"
)

type clairClient struct {
	endpoint string
}

func (cc *clairClient) analyzeRemoteImage(registryURL string, image *v1.Image, layers []string, headers map[string]string) error {
	var prevLayer string
	for _, layer := range layers {
		fullURL, err := urlfmt.FullyQualifiedURL(registryURL, url.Values{}, "v2", image.GetRemote(), "blobs", layer)
		if err != nil {
			return err
		}
		if err := cc.analyzeLayer(fullURL, layer, prevLayer, headers); err != nil {
			return err
		}
		prevLayer = layer
	}
	return nil
}

func (cc *clairClient) analyzeLayer(path, layerName, parentLayerName string, h map[string]string) error {
	payload := clairV1.LayerEnvelope{
		Layer: &clairV1.Layer{
			Name:       layerName,
			Path:       path,
			ParentName: parentLayerName,
			Format:     "Docker",
			Headers:    h,
		},
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	fullURL, err := urlfmt.FullyQualifiedURL(cc.endpoint, url.Values{}, "v1", "layers")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Minute,
	}
	log.Infof("Pushing layer: %v. This may take a while...", path)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 201 {
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Got response %d with message %s", response.StatusCode, string(body))
	}
	return nil
}

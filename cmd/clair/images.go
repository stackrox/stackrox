package main

import (
	"context"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/deckarep/golang-set"
	"github.com/docker/docker/registry"
)

func hasNecessaryAuth(images []*v1.Image) (authenticated bool) {
	registrySet := mapset.NewSet()
	for _, image := range images {
		if image.GetName().GetRegistry() != registry.IndexName {
			registrySet.Add(image.GetName().GetRegistry())
		} else if !strings.HasPrefix(image.GetName().GetRemote(), "library") {
			registrySet.Add(registry.DefaultV2Registry.Host)
		}
	}

	authenticated = true
	for registry := range registrySet.Iter() {
		if _, ok := registryAuth[registry.(string)]; !ok {
			authenticated = false
			log.Errorf("Authentication needed for registry '%v'. Please `docker login '%v'`", registry, registry)
		}
	}
	return
}

func getImages(endpoint string) ([]*v1.Image, error) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(endpoint)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := v1.NewImageServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	imagesResp, err := client.GetImages(ctx, &v1.RawQuery{})
	if err != nil {
		return nil, err
	}
	imageSet := mapset.NewSet()
	var images []*v1.Image
	for _, i := range imagesResp.GetImages() {
		if !imageSet.Contains(i.GetName().GetFullName()) {
			imageSet.Add(i.GetName().GetFullName())
			images = append(images, i)
		}
	}
	return images, nil
}

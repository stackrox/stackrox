package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/clairify"
)

// DefaultImageIntegrations are the default public registries
var DefaultImageIntegrations = []*storage.ImageIntegration{
	{
		Id:         "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
		Name:       "Public DockerHub",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "registry-1.docker.io",
			},
		},
	},
	{
		Id:         "c6a1a26d-8947-4cb0-a50d-a018856f9390",
		Name:       "Public Kubernetes GCR",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "k8s.gcr.io",
			},
		},
	},
	{
		Id:         "05fea766-e2f8-44b3-9959-eaa61a4f7466",
		Name:       "Public GCR",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "gcr.io",
			},
		},
	},
	{
		Id:         "e50087f1-6840-4d15-aeca-21ba636f0878",
		Name:       "Public Quay.io",
		Type:       "quay",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Quay{
			Quay: &storage.QuayConfig{
				Endpoint: "quay.io",
			},
		},
	},
}

// DelayedIntegration is a default integration to be added only when the trigger function returns true
type DelayedIntegration struct {
	Trigger     func() bool
	Integration *storage.ImageIntegration
}

var (
	defaultScanner = &storage.ImageIntegration{
		Id:         "169b0d3f-8277-4900-bbce-1127077defae",
		Name:       "Stackrox Scanner",
		Type:       "clairify",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER},
		IntegrationConfig: &storage.ImageIntegration_Clairify{
			Clairify: &storage.ClairifyConfig{
				Endpoint: "https://scanner.stackrox:8080",
			},
		},
	}
	// DelayedIntegrations are default integrations to be added only when the trigger function returns true
	DelayedIntegrations = []DelayedIntegration{
		{
			Trigger: func() bool {
				_, creator := clairify.Creator(nil)
				scanner, err := creator(defaultScanner)
				if err != nil {
					return false
				}
				return scanner.Test() == nil
			},
			Integration: defaultScanner,
		},
	}
)

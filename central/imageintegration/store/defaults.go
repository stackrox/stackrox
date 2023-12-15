package store

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
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
		Name:       "Public Kubernetes GCR (deprecated)",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "k8s.gcr.io",
			},
		},
	},
	{
		Id:         "f6ce8982-1a75-4430-96f3-9b22b4b66604",
		Name:       "Public Kubernetes Registry",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "registry.k8s.io",
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
		Id:         "6d7fc3f3-03d0-4b61-bf9f-34982a77bd56",
		Name:       "Public GKE GCR",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "gke.gcr.io",
			},
		},
		SkipTestIntegration: true, // /v2 endpoint is not implemented
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
	{
		Id:         "4b36a1c3-2d6f-452e-a70f-6c388a0ff947",
		Name:       "Public Microsoft Container Registry",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "mcr.microsoft.com",
			},
		},
	},
	{
		Id:         "5febb194-a21d-4109-9fad-6880dd632adc",
		Name:       "Public Amazon ECR",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "public.ecr.aws",
			},
		},
	},
	{
		Id:         "54107745-5717-49c1-9073-a2b72f7a3b49",
		Name:       "registry.access.redhat.com",
		Type:       "rhel",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "registry.access.redhat.com",
			},
		},
	},
}

// DefaultScannerV4Integration is the default Scanner V4 integration.
var DefaultScannerV4Integration = &storage.ImageIntegration{
	Id:   "a87471e6-9678-4e66-8348-91e302b6de07",
	Name: "ScannerV4",
	Type: scannerv4.TypeString,
	Categories: []storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_SCANNER,
	},
	IntegrationConfig: &storage.ImageIntegration_ScannerV4{
		ScannerV4: &storage.ScannerV4Config{
			// Use integration default values.
		},
	},
}

// DelayedIntegration is a default integration to be added only when the trigger function returns true
type DelayedIntegration struct {
	Trigger     func() bool
	Integration *storage.ImageIntegration
}

func makeDelayedIntegration(imageIntegration *storage.ImageIntegration, creatorFactory func() scanners.Creator) DelayedIntegration {
	return DelayedIntegration{
		Integration: imageIntegration,
		Trigger: func() bool {
			creator := creatorFactory()
			scanner, err := creator(imageIntegration)
			if err != nil {
				return false
			}
			return scanner.Test() == nil
		},
	}
}

var (
	defaultScanner = &storage.ImageIntegration{
		Id:   "169b0d3f-8277-4900-bbce-1127077defae",
		Name: "Stackrox Scanner",
		Type: "clairify",
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_SCANNER,
			storage.ImageIntegrationCategory_NODE_SCANNER,
		},
		IntegrationConfig: &storage.ImageIntegration_Clairify{
			Clairify: &storage.ClairifyConfig{
				Endpoint: fmt.Sprintf("https://%s:8080", clairify.GetScannerEndpoint()),
			},
		},
	}

	// DelayedIntegrations are default integrations to be added only when the trigger function returns true
	DelayedIntegrations = []DelayedIntegration{
		makeDelayedIntegration(defaultScanner, func() scanners.Creator {
			_, creator := clairify.Creator(nil)
			return creator
		}),
	}
)

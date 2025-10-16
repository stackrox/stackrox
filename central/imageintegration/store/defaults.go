package store

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
)

// DefaultImageIntegrations are the default public registries
var DefaultImageIntegrations = []*storage.ImageIntegration{
	storage.ImageIntegration_builder{
		Id:         "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
		Name:       "Public DockerHub",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "registry-1.docker.io",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "c6a1a26d-8947-4cb0-a50d-a018856f9390",
		Name:       "Public Kubernetes GCR (deprecated)",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "k8s.gcr.io",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "f6ce8982-1a75-4430-96f3-9b22b4b66604",
		Name:       "Public Kubernetes Registry",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "registry.k8s.io",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "05fea766-e2f8-44b3-9959-eaa61a4f7466",
		Name:       "Public GCR",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "gcr.io",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "6d7fc3f3-03d0-4b61-bf9f-34982a77bd56",
		Name:       "Public GKE GCR",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "gke.gcr.io",
		}.Build(),
		SkipTestIntegration: true, // /v2 endpoint is not implemented
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "e50087f1-6840-4d15-aeca-21ba636f0878",
		Name:       "Public Quay.io",
		Type:       registryTypes.QuayType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Quay: storage.QuayConfig_builder{
			Endpoint: "quay.io",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "4b36a1c3-2d6f-452e-a70f-6c388a0ff947",
		Name:       "Public Microsoft Container Registry",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "mcr.microsoft.com",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "5febb194-a21d-4109-9fad-6880dd632adc",
		Name:       "Public Amazon ECR",
		Type:       registryTypes.DockerType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "public.ecr.aws",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "54107745-5717-49c1-9073-a2b72f7a3b49",
		Name:       "registry.access.redhat.com",
		Type:       registryTypes.RedHatType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "registry.access.redhat.com",
		}.Build(),
	}.Build(),
	storage.ImageIntegration_builder{
		Id:         "48a1b014-fa42-4e3f-b45d-518c3b129f2e",
		Name:       "Public GitHub Container Registry",
		Type:       registryTypes.GHCRType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		Docker: storage.DockerConfig_builder{
			Endpoint: "ghcr.io",
		}.Build(),
		SkipTestIntegration: true, // /v2 endpoint requires authentication.
	}.Build(),
}

// DefaultScannerV4Integration is the default Scanner V4 integration.
var DefaultScannerV4Integration = &storage.ImageIntegration{
	Id:   "a87471e6-9678-4e66-8348-91e302b6de07",
	Name: "Scanner V4",
	Type: scannerTypes.ScannerV4,
	Categories: []storage.ImageIntegrationCategory{
		storage.ImageIntegrationCategory_SCANNER,
		storage.ImageIntegrationCategory_NODE_SCANNER,
	},
	IntegrationConfig: &storage.ImageIntegration_ScannerV4{
		ScannerV4: &storage.ScannerV4Config{
			IndexerEndpoint: scannerv4.DefaultIndexerEndpoint,
			MatcherEndpoint: scannerv4.DefaultMatcherEndpoint,
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
		Type: scannerTypes.Clairify,
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

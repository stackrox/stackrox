package service

import (
	"context"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/detection"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/enricher"
)

// Service provides the interface for running detection on images and containers.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DetectionServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(clusters clusterDatastore.DataStore, imageEnricher enricher.ImageEnricher, imageDatastore imageDatastore.DataStore, deploymentEnricher enrichment.Enricher, buildTimeDetector buildTimeDetection.Detector, detector deploytime.Detector, policySet detection.PolicySet) Service {
	return &serviceImpl{
		clusters:           clusters,
		imageEnricher:      imageEnricher,
		imageDatastore:     imageDatastore,
		deploymentEnricher: deploymentEnricher,
		buildTimeDetector:  buildTimeDetector,
		detector:           detector,
		policySet:          policySet,
	}
}

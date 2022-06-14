package service

import (
	"context"

	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/detection"
	buildTimeDetection "github.com/stackrox/stackrox/central/detection/buildtime"
	"github.com/stackrox/stackrox/central/detection/deploytime"
	"github.com/stackrox/stackrox/central/enrichment"
	imageDatastore "github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/notifier/processor"
	"github.com/stackrox/stackrox/central/risk/manager"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/images/enricher"
)

// Service provides the interface for running detection on images and containers.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DetectionServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(clusters clusterDatastore.DataStore, imageEnricher enricher.ImageEnricher, imageDatastore imageDatastore.DataStore, riskManager manager.Manager,
	deploymentEnricher enrichment.Enricher, buildTimeDetector buildTimeDetection.Detector, notifications processor.Processor, detector deploytime.Detector, policySet detection.PolicySet) Service {
	return &serviceImpl{
		clusters:           clusters,
		imageEnricher:      imageEnricher,
		imageDatastore:     imageDatastore,
		riskManager:        riskManager,
		deploymentEnricher: deploymentEnricher,
		buildTimeDetector:  buildTimeDetector,
		detector:           detector,
		policySet:          policySet,
		notifications:      notifications,
	}
}

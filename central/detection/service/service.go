package service

import (
	"context"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/detection"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/enhancement"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/notifier"
)

// Service provides the interface for running detection on images and containers.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DetectionServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(
	clusters clusterDatastore.DataStore,
	imageEnricher enricher.ImageEnricher,
	imageDatastore imageDatastore.DataStore,
	riskManager manager.Manager,
	deploymentEnricher enrichment.Enricher,
	buildTimeDetector buildTimeDetection.Detector,
	notifications notifier.Processor,
	detector deploytime.Detector,
	policySet detection.PolicySet,
	clusterSACHelper sachelper.ClusterSacHelper,
	connManager connection.Manager,
	watcher *enhancement.Watcher,
) Service {
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
		clusterSACHelper:   clusterSACHelper,
		connManager:        connManager,
		enhancementWatcher: watcher,
	}
}

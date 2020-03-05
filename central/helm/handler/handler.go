package handler

import (
	"net/http"

	clusterService "github.com/stackrox/rox/central/cluster/service"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Handler returns a handler for the add a cluster when using Helm, and getting back certificates for that cluster
func Handler(i siDataStore.DataStore, clusterSvc clusterService.Service) http.Handler {
	return helmHandler{
		identityStore:  i,
		clusterService: clusterSvc,
	}
}

type helmHandler struct {
	identityStore  siDataStore.DataStore
	clusterService clusterService.Service
}

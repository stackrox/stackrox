package datastore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stackrox/rox/central/metrics"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/process/id"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type datastoreImpl struct {
	storage            store.Store
	pool               postgres.DB
}

var (
	rcSAC = sac.ForResource(resources.Administration)
	log     = logging.LoggerForModule()
)

func newDatastoreImpl(
	storage store.Store,
	pool postgres.DB,
) *datastoreImpl {
	return &datastoreImpl{
		storage:            storage,
		pool:               pool,
	}
}

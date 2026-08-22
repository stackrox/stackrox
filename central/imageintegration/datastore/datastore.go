package datastore

  import (
      "context"
      "testing"

      adminEventsDS "github.com/stackrox/rox/central/administration/events/datastore"
      "github.com/stackrox/rox/central/imageintegration/store"
      pgStore "github.com/stackrox/rox/central/imageintegration/store/postgres"
      v1 "github.com/stackrox/rox/generated/api/v1"
      "github.com/stackrox/rox/generated/storage"
      "github.com/stackrox/rox/pkg/logging"
      "github.com/stackrox/rox/pkg/postgres"
      pkgSearch "github.com/stackrox/rox/pkg/search"
  )

  var (
      log = logging.LoggerForModule()
  )

  type DataStore interface {
      GetImageIntegration(ctx context.Context, id string) (*storage.ImageIntegration, bool, error)
      GetImageIntegrations(ctx context.Context, integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)

      AddImageIntegration(ctx context.Context, integration *storage.ImageIntegration) (string, error)
      UpdateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error
      RemoveImageIntegration(ctx context.Context, id string) error
      Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
      SearchImageIntegrations(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
      Count(ctx context.Context, q *v1.Query) (int, error)
  }

  func New(imageIntegrationStorage store.Store, adminEvents adminEventsDS.DataStore) DataStore {
      ds := &datastoreImpl{
          storage:     imageIntegrationStorage,
          adminEvents: adminEvents,
      }
      return ds
  }

  func NewForTestOnly(imageIntegrationStorage store.Store, adminEvents adminEventsDS.DataStore) DataStore {
      ds := &datastoreImpl{
          storage:     imageIntegrationStorage,
          adminEvents: adminEvents,
      }
      return ds
  }

  func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
      store := pgStore.New(pool)
      return New(store, adminEventsDS.GetTestPostgresDataStore(t, pool))
  }

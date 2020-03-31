package m29to30

import (
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	namespaces := []*storage.NamespaceMetadata{
		{
			Id:        "id-1",
			Name:      "name-1",
			ClusterId: "cluster-1",
		},
		{
			Id:        "id-2",
			Name:      "name-1",
			ClusterId: "cluster-2",
		},
	}

	cases := []struct {
		alert               *storage.Alert
		expectedNamespaceID string
	}{
		{
			alert: &storage.Alert{
				Id: "alert-1",
				Deployment: &storage.Alert_Deployment{
					Namespace: "name-1",
					ClusterId: "cluster-1",
				},
			},
			expectedNamespaceID: "id-1",
		},
		{
			alert: &storage.Alert{
				Id: "alert-2",
				Deployment: &storage.Alert_Deployment{
					Namespace: "name-1",
					ClusterId: "cluster-2",
				},
			},
			expectedNamespaceID: "id-2",
		},
	}

	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)

	badgerDB, err := badgerhelpers.NewTemp("temp")
	require.NoError(t, err)

	err = createNamespaceBucket(db)
	require.NoError(t, err)
	err = fillNamespaces(db, namespaces)
	require.NoError(t, err)

	err = fillAlerts(badgerDB, []*storage.Alert{cases[0].alert, cases[1].alert})
	require.NoError(t, err)

	require.NoError(t, updateAlertDeploymentWithNamespaceID(&types.Databases{BoltDB: db, BadgerDB: badgerDB}))

	for _, c := range cases {
		validateMigration(t, badgerDB, c.alert, c.expectedNamespaceID)
	}
}

func createNamespaceBucket(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(namespaceBucketName))
		return err
	})
}

func fillNamespaces(db *bbolt.DB, namespaces []*storage.NamespaceMetadata) error {
	nsBucket := bolthelpers.TopLevelRef(db, namespaceBucketName)
	for _, namespace := range namespaces {
		err := nsBucket.Update(func(b *bbolt.Bucket) error {
			data, err := proto.Marshal(namespace)
			if err != nil {
				return err
			}
			return b.Put([]byte(namespace.GetId()), data)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func fillAlerts(db *badger.DB, alerts []*storage.Alert) error {
	for _, alert := range alerts {
		err := db.Update(func(tx *badger.Txn) error {
			key := getAlertKey(alert.GetId())

			data, err := proto.Marshal(alert)
			if err != nil {
				return err
			}
			return tx.Set(key, data)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validateMigration(t *testing.T, db *badger.DB, alert *storage.Alert, expectedNamespaceID string) {
	err := db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(getAlertKey(alert.GetId()))
		require.NoError(t, err)

		alert = &storage.Alert{}
		err = item.Value(func(v []byte) error {
			return proto.Unmarshal(v, alert)
		})
		require.NoError(t, err)

		assert.Equal(t, expectedNamespaceID, alert.GetDeployment().GetNamespaceId())
		return nil
	})
	require.NoError(t, err)
}

func getAlertKey(alertID string) []byte {
	key := make([]byte, 0, len(alertBucketName)+len(alertID))
	key = append(key, alertBucketName...)
	key = append(key, alertID...)

	return key
}

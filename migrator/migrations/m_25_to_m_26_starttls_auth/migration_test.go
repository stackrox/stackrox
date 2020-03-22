package m25tom26

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)

	randomNotifier := &storage.Notifier{
		Id:   "1",
		Name: "random type",
		Type: "random",
	}

	emailNoStartTLS := &storage.Notifier{
		Id:   "2",
		Name: "Email with no start tls",
		Type: "email",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				DEPRECATEDUseStartTLS: false,
			}},
	}

	emailWithStartTLS := &storage.Notifier{
		Id:   "3",
		Name: "Email with start tls",
		Type: "email",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				DEPRECATEDUseStartTLS: true,
				DisableTLS:            true,
			}},
	}

	expectedNewEmailWithStartTLS := &storage.Notifier{
		Id:   "3",
		Name: "Email with start tls",
		Type: "email",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				DEPRECATEDUseStartTLS: true,
				StartTLSAuthMethod:    storage.Email_PLAIN,
				DisableTLS:            true,
			}},
	}

	emailWithStartTLSAndTLS := &storage.Notifier{
		Id:   "4",
		Name: "Email with start tls and tls",
		Type: "email",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				DEPRECATEDUseStartTLS: true,
			}},
	}

	expectedEmailWithStartTLSAndTLS := &storage.Notifier{
		Id:   "4",
		Name: "Email with start tls and tls",
		Type: "email",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				DEPRECATEDUseStartTLS: false,
			}},
	}

	notifiers := []*storage.Notifier{
		randomNotifier,
		emailNoStartTLS,
		emailWithStartTLS,
		emailWithStartTLSAndTLS,
	}
	expectedNotifiers := []*storage.Notifier{
		randomNotifier,
		emailNoStartTLS,
		expectedNewEmailWithStartTLS,
		expectedEmailWithStartTLSAndTLS,
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(notifierBucket)
		require.NoError(t, err)
		for _, n := range notifiers {
			bytes, err := proto.Marshal(n)
			require.NoError(t, err)
			require.NoError(t, bucket.Put([]byte(n.GetId()), bytes))
		}
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, migrateEmail(db))

	var actualNotifiers []*storage.Notifier
	err = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		require.NotNil(t, bucket)

		return bucket.ForEach(func(k, v []byte) error {
			var n storage.Notifier
			require.NoError(t, proto.Unmarshal(v, &n))
			actualNotifiers = append(actualNotifiers, &n)
			return nil
		})
	})
	require.NoError(t, err)

	assert.Equal(t, expectedNotifiers, actualNotifiers)
}

package m25tom26

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 25,
		VersionAfter:   storage.Version{SeqNum: 26},
		Run: func(databases *types.Databases) error {
			return migrateEmail(databases.BoltDB)
		},
	}

	notifierBucket = []byte("notifiers")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateEmail(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifierBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var notifier storage.Notifier
			if err := proto.Unmarshal(v, &notifier); err != nil {
				return err
			}
			if notifier.GetType() != "email" {
				return nil
			}
			email := notifier.GetEmail()
			if email == nil {
				return nil
			}
			// This was a valid configuration previously and the result was to just ignore the STARTTLS
			// and use TLS. This change preserves that functionality while also not allowing people to set STARTTLS
			// and TLS together in the future
			if !email.GetDisableTLS() && email.GetDEPRECATEDUseStartTLS() {
				email.DEPRECATEDUseStartTLS = false
			}
			if email.GetDEPRECATEDUseStartTLS() {
				email.StartTLSAuthMethod = storage.Email_PLAIN
			}
			newBytes, err := proto.Marshal(&notifier)
			if err != nil {
				return err
			}
			return bucket.Put(k, newBytes)
		})
	})
}

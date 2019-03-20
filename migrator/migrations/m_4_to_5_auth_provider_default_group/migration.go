package m4to5

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

const (
	// Defined in pkg/auth/authprovider/basic
	basicAuthProviderTypeName = "basic"
)

var (
	groupsBucketName        = []byte("groups")
	authProvidersBucketName = []byte("authProviders")
)

func retrieveAuthProviderIDsExceptBasic(db *bolt.DB) (authProviderIDs []string, err error) {
	authProvidersBucket := bolthelpers.TopLevelRef(db, authProvidersBucketName)
	err = authProvidersBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			authProvider := new(storage.AuthProvider)
			if err := proto.Unmarshal(v, authProvider); err != nil {
				return fmt.Errorf("unmarshaling auth provider: %v", err)
			}
			if authProvider.GetType() != basicAuthProviderTypeName {
				authProviderIDs = append(authProviderIDs, authProvider.GetId())
			}
			return nil
		})
	})
	return
}

func findAuthProvidersNotRepresentedInGroups(db *bolt.DB, authProviderIDsToCheck []string) (notRepresented []string, err error) {
	groupsBucket := bolthelpers.TopLevelRef(db, groupsBucketName)

	err = groupsBucket.View(func(b *bolt.Bucket) error {
		c := b.Cursor()
		for _, authProviderID := range authProviderIDsToCheck {
			// groups are serialized with "authproviderid:key:value" as the key.
			prefix := []byte(fmt.Sprintf("%s:", authProviderID))
			k, _ := c.Seek(prefix)
			if k == nil || !bytes.HasPrefix(k, prefix) {
				notRepresented = append(notRepresented, authProviderID)
			}
		}
		return nil
	})
	return
}

func addDefaultAdminMappingForAuthProviders(db *bolt.DB, authProviderIDs []string) error {
	// Early exit -- nothing to do!
	if len(authProviderIDs) == 0 {
		return nil
	}

	groupsBucket := bolthelpers.TopLevelRef(db, groupsBucketName)
	return groupsBucket.Update(func(b *bolt.Bucket) error {
		for _, id := range authProviderIDs {
			// groups are serialized with "authproviderid:key:value" as the key.
			// If no key and value are specified, then all users authenticated through this auth provider
			// are given the role specified.
			// This effectively gives all users authenticated by this auth provider the Admin role.
			// See central/group/store/serialize.go for serialization logic.
			if err := b.Put([]byte(fmt.Sprintf("%s::", id)), []byte("Admin")); err != nil {
				return fmt.Errorf("inserting auth provider %q: %v", id, err)
			}
		}
		return nil
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 4,
		VersionAfter:   storage.Version{SeqNum: 5},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			authProviderIDs, err := retrieveAuthProviderIDsExceptBasic(db)
			if err != nil {
				return fmt.Errorf("retrieving auth providers: %v", err)
			}
			notRepresentedAuthProviders, err := findAuthProvidersNotRepresentedInGroups(db, authProviderIDs)
			if err != nil {
				return fmt.Errorf("finding auth providers not represented: %v", err)
			}

			err = addDefaultAdminMappingForAuthProviders(db, notRepresentedAuthProviders)
			if err != nil {
				return fmt.Errorf("adding default admin mapping: %v", err)
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

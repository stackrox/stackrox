package m16tom17

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

type migrationTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *migrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(imagesBucketName)
		return err
	}))
	suite.db = db
}

func (suite *migrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertThing(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *migrationTestSuite) mustInsertRoles(image *storage.Image) {
	rolesBucket := bolthelpers.TopLevelRef(suite.db, imagesBucketName)
	suite.NoError(insertThing(rolesBucket, image.GetId(), image))
}

func (suite *migrationTestSuite) TestPurgeDebugMetricsMigration() {
	oldImages := []*storage.Image{
		{
			Id: "sha1",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
						},
					},
				},
			},
		},
		{
			Id: "sha2",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
							{
								Cve: "derp3",
							},
						},
					},
				},
			},
		},
		{
			Id: "sha3",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp0",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
						},
					},
				},
			},
		},
	}

	expectedImages := []*storage.Image{
		{
			Id: "sha1",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
						},
					},
				},
			},
			SetComponents: &storage.Image_Components{
				Components: 1,
			},
			SetCves: &storage.Image_Cves{
				Cves: 2,
			},
			SetFixable: &storage.Image_FixableCves{
				FixableCves: 1,
			},
		},
		{
			Id: "sha2",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
							{
								Cve: "derp3",
							},
						},
					},
				},
			},
			SetComponents: &storage.Image_Components{
				Components: 1,
			},
			SetCves: &storage.Image_Cves{
				Cves: 3,
			},
			SetFixable: &storage.Image_FixableCves{
				FixableCves: 1,
			},
		},
		{
			Id: "sha3",
			Scan: &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Vulns: []*storage.EmbeddedVulnerability{
							{
								Cve: "derp0",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp",
								SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
									FixedBy: "v1.2",
								},
							},
							{
								Cve: "derp2",
							},
						},
					},
				},
			},
			SetComponents: &storage.Image_Components{
				Components: 1,
			},
			SetCves: &storage.Image_Cves{
				Cves: 3,
			},
			SetFixable: &storage.Image_FixableCves{
				FixableCves: 2,
			},
		},
	}

	for _, image := range oldImages {
		suite.mustInsertRoles(image)
	}

	suite.NoError(migration.Run(&types.Databases{BoltDB: suite.db}))

	newImages := make([]*storage.Image, 0, len(oldImages))
	imagesBucket := bolthelpers.TopLevelRef(suite.db, imagesBucketName)
	suite.NoError(imagesBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var image storage.Image
			err := proto.Unmarshal(v, &image)
			if err != nil {
				return err
			}
			newImages = append(newImages, &image)
			return nil
		})
	}))
	suite.ElementsMatch(expectedImages, newImages)
}

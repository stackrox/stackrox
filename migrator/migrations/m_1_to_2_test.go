package migrations

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMigration1to2(t *testing.T) {
	suite.Run(t, new(Migration1To2TestSuite))
}

type Migration1To2TestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *Migration1To2TestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite.Suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(alertsBucket)
		return err
	}))

	suite.db = db
}

func (suite *Migration1To2TestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func getNormalAlert(id string) *storage.Alert {
	return &storage.Alert{
		Id: id,
		Violations: []*storage.Alert_Violation{
			{Message: "Blah", Link: "baaa"},
			{Message: "Blah2", Link: "baaaa2"},
		},
	}
}

func getProcesses() []*storage.ProcessIndicator {
	return []*storage.ProcessIndicator{
		{Id: "PROCESS1", Signal: &storage.ProcessSignal{Name: "apt-get"}},
		{Id: "PROCESS2"},
	}
}

func getLegacyAlertWithProcesses(id string) *storage.Alert {
	alert := getNormalAlert(id)
	alert.Violations = append(alert.Violations, &storage.Alert_Violation{
		Message:             "ProcessBlah",
		DEPRECATEDProcesses: getProcesses(),
	})
	return alert
}

func getNewStyleAlertWithProcesses(id string) *storage.Alert {
	alert := getNormalAlert(id)
	alert.ProcessViolation = &storage.Alert_ProcessViolation{
		Message:   "ProcessBlah",
		Processes: getProcesses(),
	}
	return alert
}

func (suite *Migration1To2TestSuite) mustGetAlert(id string) *storage.Alert {
	bucketRef := bolthelpers.TopLevelRef(suite.db, alertsBucket)
	alert := new(storage.Alert)
	suite.NoError(bucketRef.View(func(b *bolt.Bucket) error {
		bytes := b.Get([]byte(id))
		err := proto.Unmarshal(bytes, alert)
		if err != nil {
			return err
		}
		return nil
	}))
	return alert
}

func (suite *Migration1To2TestSuite) mustInsertAlert(alert *storage.Alert) {
	bucketRef := bolthelpers.TopLevelRef(suite.db, alertsBucket)
	suite.NoError(bucketRef.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(alert)
		if err != nil {
			return err
		}
		err = b.Put([]byte(alert.GetId()), bytes)
		if err != nil {
			return err
		}
		return nil
	}))
}

func (suite *Migration1To2TestSuite) TestWithOnlyNormalAlerts() {
	ids := []string{"id1", "id2", "id3"}

	for _, id := range ids {
		suite.mustInsertAlert(getNormalAlert(id))
	}

	suite.NoError(alertViolationMigration.Run(suite.db))

	for _, id := range ids {
		got := suite.mustGetAlert(id)
		suite.Equal(getNormalAlert(id), got)
	}
}

func (suite *Migration1To2TestSuite) TestWithProcessAlerts() {
	processIDs := []string{"processid1", "processid2"}
	for _, id := range processIDs {
		suite.mustInsertAlert(getLegacyAlertWithProcesses(id))
	}

	suite.NoError(alertViolationMigration.Run(suite.db))

	for _, id := range processIDs {
		got := suite.mustGetAlert(id)
		suite.Equal(getNewStyleAlertWithProcesses(id), got)
	}
}

func (suite *Migration1To2TestSuite) TestWithBothKindsOfAlerts() {
	ids := []string{"id1", "id2", "id3"}
	processIDs := []string{"processid1", "processid2"}
	for _, id := range ids {
		suite.mustInsertAlert(getNormalAlert(id))
	}
	for _, id := range processIDs {
		suite.mustInsertAlert(getLegacyAlertWithProcesses(id))
	}

	suite.NoError(alertViolationMigration.Run(suite.db))
	for _, id := range processIDs {
		got := suite.mustGetAlert(id)
		suite.Equal(getNewStyleAlertWithProcesses(id), got)
	}
}

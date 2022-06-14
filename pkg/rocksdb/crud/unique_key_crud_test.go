package generic

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stretchr/testify/suite"
)

func uniqueKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Alert).GetDeployment().GetName())
}

func getAlertWithDeploymentName(id, deploymentName string) *storage.Alert {
	a := fixtures.GetAlertWithID(id)
	a.GetDeployment().Name = deploymentName
	return a
}

func TestUniqueKeyCRUD(t *testing.T) {
	suite.Run(t, new(UniqueKeyCRUDTestSuite))
}

type UniqueKeyCRUDTestSuite struct {
	suite.Suite

	dir string
	db  *rocksdb.RocksDB

	crud db.Crud
}

func (s *UniqueKeyCRUDTestSuite) SetupTest() {
	var err error
	dir := s.T().TempDir()

	s.dir = dir

	s.db, err = rocksdb.New(dir)
	s.NoError(err)
	s.crud = NewUniqueKeyCRUD(s.db, []byte("bucket"), alertKeyFunc, alloc, uniqueKeyFunc, false)
}

func (s *UniqueKeyCRUDTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *UniqueKeyCRUDTestSuite) TestUpsert() {
	alert1 := fixtures.GetAlertWithID("alert1")
	alert1.GetDeployment().Name = "dep1"

	alert2 := fixtures.GetAlertWithID("alert2")
	alert2.GetDeployment().Name = "dep2"

	// Insert both alerts successfully
	s.NoError(s.crud.Upsert(alert1))
	s.NoError(s.crud.Upsert(alert2))

	// Upsert the alerts again. There should not be a collision with itself
	s.NoError(s.crud.Upsert(alert1))
	s.NoError(s.crud.Upsert(alert2))

	// Insert alert3 with the same unique key as 1
	alert3 := fixtures.GetAlertWithID("alert3")
	alert3.GetDeployment().Name = "dep1"

	// Should have conflict error
	s.Error(s.crud.Upsert(alert3))

	// Delete alert1 and the upsert alert3 and it should now succeed
	s.NoError(s.crud.Delete(alert1.GetId()))
	s.NoError(s.crud.Upsert(alert3))
}

func (s *UniqueKeyCRUDTestSuite) TestUpsertMany() {
	alert1 := getAlertWithDeploymentName("alert1", "dep1")
	alert2 := getAlertWithDeploymentName("alert2", "dep2")
	alert3 := getAlertWithDeploymentName("alert3", "dep1")

	// Conflict between batch
	s.Error(s.crud.UpsertMany([]proto.Message{alert1, alert2, alert3}))

	// No conflicts
	s.NoError(s.crud.UpsertMany([]proto.Message{alert1, alert2}))

	// alert3 conflicts with existing alert1 in DB
	s.Error(s.crud.UpsertMany([]proto.Message{alert3}))

	s.NoError(s.crud.Delete(alert1.GetId()))
	s.NoError(s.crud.UpsertMany([]proto.Message{alert3}))
}

func (s *UniqueKeyCRUDTestSuite) TestUpsertWithID() {
	alert1 := fixtures.GetAlertWithID("noop1")
	alert1.GetDeployment().Name = "dep1"

	alert2 := fixtures.GetAlertWithID("noop1")
	alert2.GetDeployment().Name = "dep2"

	// Insert both alerts successfully
	s.NoError(s.crud.UpsertWithID("alert1", alert1))
	_, exists, err := s.crud.Get("alert1")
	s.NoError(err)
	s.True(exists)

	s.NoError(s.crud.UpsertWithID("alert2", alert2))

	// Upsert the alerts again. There should not be a collision with itself
	s.NoError(s.crud.UpsertWithID("alert1", alert1))
	s.NoError(s.crud.UpsertWithID("alert2", alert2))

	// Insert alert3 with the same unique key as 1
	alert3 := fixtures.GetAlertWithID("noop1")
	alert3.GetDeployment().Name = "dep1"

	// Should have conflict error
	s.Error(s.crud.UpsertWithID("alert3", alert3))

	// Delete alert1 and the upsert alert3 and it should now succeed
	s.NoError(s.crud.Delete("alert1"))
	s.NoError(s.crud.UpsertWithID("alert3", alert3))
}

func (s *UniqueKeyCRUDTestSuite) TestUpsertManyWithIDs() {
	alert1 := getAlertWithDeploymentName("alert1", "dep1")
	alert2 := getAlertWithDeploymentName("alert2", "dep2")
	alert3 := getAlertWithDeploymentName("alert3", "dep1")

	// Conflict between batch
	s.Error(s.crud.UpsertManyWithIDs([]string{alert1.GetId(), alert2.GetId(), alert3.GetId()}, []proto.Message{alert1, alert2, alert3}))

	// No conflicts
	s.NoError(s.crud.UpsertManyWithIDs([]string{alert1.GetId(), alert2.GetId()}, []proto.Message{alert1, alert2}))

	// alert3 conflicts with existing alert1 in DB
	s.Error(s.crud.UpsertManyWithIDs([]string{alert3.GetId()}, []proto.Message{alert3}))

	s.NoError(s.crud.Delete(alert1.GetId()))
	s.NoError(s.crud.UpsertManyWithIDs([]string{alert3.GetId()}, []proto.Message{alert3}))
}

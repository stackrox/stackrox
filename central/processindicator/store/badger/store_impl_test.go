package badger

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
)

func TestIndicatorStore(t *testing.T) {
	if features.ManagedDB.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(IndicatorStoreTestSuite))
}

type IndicatorStoreTestSuite struct {
	suite.Suite

	db  *badger.DB
	dir string

	store store.Store
}

func (suite *IndicatorStoreTestSuite) SetupSuite() {
	db, dir, err := badgerhelper.NewTemp(suite.T().Name()+".db", false)
	if err != nil {
		suite.FailNow("Failed to make BadgerDB", err.Error())
	}

	suite.db = db
	suite.dir = dir
	suite.store = New(db)
}

func (suite *IndicatorStoreTestSuite) TearDownSuite() {
	_ = suite.db.Close()
	_ = os.RemoveAll(suite.dir)
}

func (suite *IndicatorStoreTestSuite) verifyIndicatorsAre(indicators ...*storage.ProcessIndicator) {
	for _, i := range indicators {
		retrievedIndicator, exists, err := suite.store.GetProcessIndicator(i.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.NotNil(retrievedIndicator)
		suite.Equal(i, retrievedIndicator)
	}

	// Get all indicators
	retrievedIndicators, err := suite.store.GetProcessIndicators()
	suite.NoError(err)
	suite.ElementsMatch(indicators, retrievedIndicators)

}

func (suite *IndicatorStoreTestSuite) TestIndicators() {
	repeatedSignal := &storage.ProcessSignal{
		Args:         "da_args",
		ContainerId:  "aa",
		Name:         "blah",
		ExecFilePath: "file",
	}

	indicators := []*storage.ProcessIndicator{
		{
			Id:           "id1",
			DeploymentId: "d1",

			Signal: repeatedSignal,
		},
		{
			Id:           "id2",
			DeploymentId: "d2",

			Signal: &storage.ProcessSignal{
				Args: "args2",
			},
		},
	}

	repeatIndicator := &storage.ProcessIndicator{
		Id:           "id3",
		DeploymentId: "d1",
		Signal:       repeatedSignal,
	}

	for _, i := range indicators {
		tx, err := suite.store.GetTxnCount()
		suite.NoError(err)
		_, err = suite.store.AddProcessIndicator(i)
		suite.NoError(err)

		newTx, err := suite.store.GetTxnCount()
		suite.NoError(err)
		suite.Equal(tx+1, newTx)
	}

	suite.verifyIndicatorsAre(indicators...)

	var walkedIndicators []*storage.ProcessIndicator
	err := suite.store.WalkAll(func(pi *storage.ProcessIndicator) error {
		walkedIndicators = append(walkedIndicators, pi)
		return nil
	})
	suite.NoError(err)
	suite.verifyIndicatorsAre(walkedIndicators...)

	// Adding an indicator with the same secondary key should replace the original one.
	tx, err := suite.store.GetTxnCount()
	suite.NoError(err)
	removed, err := suite.store.AddProcessIndicator(repeatIndicator)
	suite.NoError(err)
	suite.Equal("id1", removed)
	suite.verifyIndicatorsAre(indicators[1], repeatIndicator)

	newTx, err := suite.store.GetTxnCount()
	suite.NoError(err)
	suite.Equal(tx+2, newTx)

	for _, i := range []*storage.ProcessIndicator{indicators[1], repeatIndicator} {
		tx, err := suite.store.GetTxnCount()
		suite.NoError(err)

		suite.NoError(suite.store.RemoveProcessIndicator(i.GetId()))

		newTx, err := suite.store.GetTxnCount()
		suite.NoError(err)
		suite.Equal(tx+1, newTx)
	}
	suite.verifyIndicatorsAre()

	tx, err = suite.store.GetTxnCount()
	suite.NoError(err)

	oldIDs, err := suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)
	suite.Empty(oldIDs)

	newTx, err = suite.store.GetTxnCount()
	suite.NoError(err)
	suite.Equal(tx+1, newTx)

	// Modify indicator ids so we can batch add and we should get the old values out
	for idx, i := range indicators {
		i.Id = fmt.Sprintf("id%d", idx+3)
	}

	tx, err = suite.store.GetTxnCount()
	suite.NoError(err)

	oldIDs, err = suite.store.AddProcessIndicators(indicators...)
	suite.NoError(err)
	suite.ElementsMatch([]string{"id1", "id2"}, oldIDs)

	newTx, err = suite.store.GetTxnCount()
	suite.NoError(err)
	suite.Equal(tx+2, newTx)

	indicators, err = suite.store.GetProcessIndicators()
	suite.NoError(err)

	var ids []string
	for _, i := range indicators {
		ids = append(ids, i.GetId())
	}
	tx, err = suite.store.GetTxnCount()
	suite.NoError(err)

	suite.NoError(suite.store.RemoveProcessIndicators(ids))

	newTx, err = suite.store.GetTxnCount()
	suite.NoError(err)
	suite.Equal(tx+1, newTx)

	v, err := suite.store.GetProcessInfoToArgs()
	suite.NoError(err)
	suite.Len(v, 0)
}

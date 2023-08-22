//go:build sql_integration

package manager

import (
	"bytes"
	"context"
	"hash/crc32"
	"io"
	"testing"

	blobstore "github.com/stackrox/rox/central/blob/datastore"
	blobSearch "github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	blobPg "github.com/stackrox/rox/central/blob/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

const (
	validFilePath   = "1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko.gz"
	invalidFilePath = "1123dde0458e7a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko"
)

type managerTestSuite struct {
	suite.Suite

	mgr       *manager
	testDB    *pgtest.TestPostgres
	store     store.Store
	datastore blobstore.Datastore
}

func TestManager(t *testing.T) {
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) SetupTest() {
	s.testDB = pgtest.ForT(s.T())
	s.store = store.New(s.testDB.DB)
	searcher := blobSearch.New(s.store, blobPg.NewIndexer(s.testDB.DB))
	s.datastore = blobstore.NewDatastore(s.store, searcher)
	s.mgr = newManager(s.datastore)
	s.mgr.freeStorageThreshold = 0 // not interested in testing this
}

func (s *managerTestSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}

func (s *managerTestSuite) TestGetExistingProbeFilesWithNoProbes() {
	s.Require().NoError(s.mgr.Initialize())

	allAccessCtx := sac.WithAllAccess(context.Background())
	fileInfos, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{validFilePath})
	s.NoError(err)
	s.Empty(fileInfos)
}

func (s *managerTestSuite) TestGetExistingProbeFilesWithInvalidPath() {
	s.Require().NoError(s.mgr.Initialize())

	allAccessCtx := sac.WithAllAccess(context.Background())
	_, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{invalidFilePath})
	s.Error(err)
}

func (s *managerTestSuite) TestStoreAndGetExistingProbeFile() {
	s.Require().NoError(s.mgr.Initialize())

	data := []byte("foobarbaz")
	crc32Sum := crc32.ChecksumIEEE(data)

	allAccessCtx := sac.WithAllAccess(context.Background())
	s.False(s.mgr.IsAvailable(allAccessCtx))
	s.Require().NoError(s.mgr.StoreFile(allAccessCtx, validFilePath, bytes.NewReader(data), int64(len(data)), crc32Sum))
	s.True(s.mgr.IsAvailable(allAccessCtx))

	fileInfos, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{validFilePath})
	s.NoError(err)
	s.Require().Len(fileInfos, 1)

	s.Equal(validFilePath, fileInfos[0].GetName())
	s.EqualValues(len("foobarbaz"), fileInfos[0].GetSize_())
	s.Equal(crc32Sum, fileInfos[0].GetCrc32())
}

func (s *managerTestSuite) TestLoadProbeFile() {
	s.Require().NoError(s.mgr.Initialize())

	data := []byte("foobarbaz")
	crc32Sum := crc32.ChecksumIEEE(data)

	allAccessCtx := sac.WithAllAccess(context.Background())
	s.Require().NoError(s.mgr.StoreFile(allAccessCtx, validFilePath, bytes.NewReader(data), int64(len(data)), crc32Sum))

	reader, length, err := s.mgr.LoadProbe(allAccessCtx, validFilePath)
	s.NoError(err)
	s.EqualValues(len(data), length)

	buf := bytes.NewBuffer(nil)
	n, err := io.Copy(buf, reader)
	s.Require().NoError(err)
	s.EqualValues(n, len(data))
	s.Equal(data, buf.Bytes())
	s.NoError(reader.Close())

	// Reboot
	s.Require().NoError(s.mgr.Initialize())
}

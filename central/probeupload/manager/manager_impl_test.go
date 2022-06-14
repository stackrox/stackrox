package manager

import (
	"bytes"
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/stackrox/pkg/binenc"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

const (
	validFilePath   = "1123dde0458e72a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko.gz"
	invalidFilePath = "1123dde0458e7a49880b06922e135dbcd36fb784fed530ab84ddfa8924e5c05/collector-4.9.24-coreos.ko"
)

type managerTestSuite struct {
	suite.Suite

	dataDir string
	mgr     *manager
}

func TestManager(t *testing.T) {
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) SetupTest() {
	s.dataDir = s.T().TempDir()
	s.mgr = newManager(s.dataDir)
	s.mgr.freeDiskThreshold = 0 // not interested in testing this
}

func (s *managerTestSuite) TestInitializeOnEmptyDir() {
	s.NoError(s.mgr.Initialize(), "initializing on empty directory should succeed")
}

func (s *managerTestSuite) TestGetExistingProbeFilesOnEmptyDir() {
	s.Require().NoError(s.mgr.Initialize())

	allAccessCtx := sac.WithAllAccess(context.Background())
	fileInfos, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{validFilePath})
	s.NoError(err)
	s.Empty(fileInfos)
}

func (s *managerTestSuite) TestGetExistingProbeFilesOnNonEmptyDir() {
	s.Require().NoError(s.mgr.Initialize())

	fileDataDir := filepath.Join(s.mgr.rootDir, filepath.FromSlash(validFilePath))
	s.Require().NoError(os.MkdirAll(fileDataDir, 0700))
	s.Require().NoError(os.WriteFile(filepath.Join(fileDataDir, dataFileName), []byte("foobarbaz"), 0600))
	s.Require().NoError(os.WriteFile(filepath.Join(fileDataDir, crc32FileName), binenc.BigEndian.EncodeUint32(1337), 0600))

	allAccessCtx := sac.WithAllAccess(context.Background())
	fileInfos, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{validFilePath})
	s.NoError(err)
	s.Require().Len(fileInfos, 1)

	s.Equal(validFilePath, fileInfos[0].GetName())
	s.EqualValues(len("foobarbaz"), fileInfos[0].GetSize_())
	s.EqualValues(1337, fileInfos[0].GetCrc32())
}

func (s *managerTestSuite) TestGetExistingProbeFilesWithInvalidPath() {
	s.Require().NoError(s.mgr.Initialize())

	allAccessCtx := sac.WithAllAccess(context.Background())
	_, err := s.mgr.GetExistingProbeFiles(allAccessCtx, []string{invalidFilePath})
	s.Error(err)
}

func (s *managerTestSuite) TestStoreFile() {
	s.Require().NoError(s.mgr.Initialize())

	data := []byte("foobarbaz")
	crc32Sum := crc32.ChecksumIEEE(data)

	allAccessCtx := sac.WithAllAccess(context.Background())
	s.Require().NoError(s.mgr.StoreFile(allAccessCtx, validFilePath, bytes.NewReader(data), int64(len(data)), crc32Sum))

	fileDataDir := s.mgr.getDataDir(validFilePath)
	_, err := os.Stat(fileDataDir)
	s.Require().NoError(err)

	dataContents, err := os.ReadFile(filepath.Join(fileDataDir, dataFileName))
	s.NoError(err)
	s.Equal(data, dataContents)

	checksumContents, err := os.ReadFile(filepath.Join(fileDataDir, crc32FileName))
	s.NoError(err)
	s.Equal(binenc.BigEndian.EncodeUint32(crc32Sum), checksumContents)
}

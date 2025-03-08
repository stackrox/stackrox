package ioutils

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	dataLen = 1337
)

type checksumReaderSuite struct {
	suite.Suite

	algo     hash.Hash
	data     []byte
	checksum []byte
}

func (s *checksumReaderSuite) SetupTest() {
	s.data = make([]byte, dataLen)
	_, err := rand.Read(s.data)
	require.NoError(s.T(), err)

	s.algo.Reset()
	n, err := s.algo.Write(s.data)
	s.Require().Equal(len(s.data), n)
	s.Require().NoError(err)

	s.checksum = s.algo.Sum(nil)
}

func (s *checksumReaderSuite) mutate(data []byte) {
	bitMask := byte(1) << (uint(rand.Int()) % 8)
	byteIdx := rand.Int() % len(data)
	data[byteIdx] = data[byteIdx] ^ bitMask
}

func (s *checksumReaderSuite) TestSuccess() {
	checksumReader, err := NewChecksumReader(bytes.NewReader(s.data), s.algo, s.checksum)
	s.Require().NoError(err)

	readBuf := make([]byte, len(s.data))
	_, err = io.ReadFull(checksumReader, readBuf)
	if err == io.EOF {
		err = nil
	}
	s.Require().NoError(err)
	s.Equal(s.data, readBuf)
	n, err := checksumReader.Read(readBuf[:1])
	s.Equal(io.EOF, err)
	s.Zero(n)
	s.NoError(checksumReader.Close())
}

func (s *checksumReaderSuite) TestSuccess32() {
	algo32, _ := s.algo.(hash.Hash32)
	if algo32 == nil {
		s.T().Skip("algo is not a 32-bit checksum algo")
	}
	algo32.Reset()
	n, err := algo32.Write(s.data)
	s.Require().NoError(err)
	s.Require().Equal(len(s.data), n)

	checksum32 := algo32.Sum32()

	checksumReader := NewChecksum32Reader(bytes.NewReader(s.data), algo32, checksum32)

	readBuf := make([]byte, len(s.data))
	_, err = io.ReadFull(checksumReader, readBuf)
	if err == io.EOF {
		err = nil
	}
	s.Require().NoError(err)
	s.Equal(s.data, readBuf)
	n, err = checksumReader.Read(readBuf[:1])
	s.Equal(io.EOF, err)
	s.Zero(n)
	s.NoError(checksumReader.Close())
}

func (s *checksumReaderSuite) TestSuccess64() {
	algo64, _ := s.algo.(hash.Hash64)
	if algo64 == nil {
		s.T().Skip("algo is not a 64-bit checksum algo")
	}
	algo64.Reset()
	n, err := algo64.Write(s.data)
	s.Require().NoError(err)
	s.Require().Equal(len(s.data), n)

	checksum64 := algo64.Sum64()

	checksumReader := NewChecksum64Reader(bytes.NewReader(s.data), algo64, checksum64)

	readBuf := make([]byte, len(s.data))
	_, err = io.ReadFull(checksumReader, readBuf)
	if err == io.EOF {
		err = nil
	}
	s.Require().NoError(err)
	s.Equal(s.data, readBuf)
	n, err = checksumReader.Read(readBuf[:1])
	s.Equal(io.EOF, err)
	s.Zero(n)
	s.NoError(checksumReader.Close())
}

func (s *checksumReaderSuite) TestDataMutation() {
	// Mutate a single bit. All checksum algos should detect this.
	s.mutate(s.data)

	checksumReader, err := NewChecksumReader(bytes.NewReader(s.data), s.algo, s.checksum)
	s.Require().NoError(err)
	readBuf := make([]byte, len(s.data))
	_, err = io.ReadFull(checksumReader, readBuf)
	s.Equal(s.data, readBuf)
	if err == nil {
		var n int
		n, err = checksumReader.Read(readBuf[:1])
		s.Zero(n)
	}
	s.Require().Error(err)
	s.NotEqual(io.EOF, err)
	s.Contains(err.Error(), "checksum")

	err = checksumReader.Close()
	s.Require().Error(err)
	s.Contains(err.Error(), "checksum")
}

func (s *checksumReaderSuite) TestChecksumMutation() {
	// Mutate a single bit in the checksum. All checksum algos MUST detect this.
	s.mutate(s.checksum)

	checksumReader, err := NewChecksumReader(bytes.NewReader(s.data), s.algo, s.checksum)
	s.Require().NoError(err)
	readBuf := make([]byte, len(s.data))
	_, err = io.ReadFull(checksumReader, readBuf)
	s.Equal(s.data, readBuf)
	if err == nil {
		var n int
		n, err = checksumReader.Read(readBuf[:2])
		s.Zero(n)
	}
	s.Require().Error(err)
	s.NotEqual(io.EOF, err)
	s.Contains(err.Error(), "checksum")

	err = checksumReader.Close()
	s.Require().Error(err)
	s.Contains(err.Error(), "checksum")
}

func (s *checksumReaderSuite) TestIncompleteRead() {
	checksumReader, err := NewChecksumReader(bytes.NewReader(s.data), s.algo, s.checksum)
	s.Require().NoError(err)

	readBuf := make([]byte, len(s.data)-1)
	_, err = io.ReadFull(checksumReader, readBuf)
	s.Equal(s.data[:len(s.data)-1], readBuf)
	s.Require().NoError(err)
	err = checksumReader.Close()
	s.Require().Error(err)
	s.Contains(err.Error(), "checksum")
}

func TestChecksumReader(t *testing.T) {
	t.Parallel()

	algos := map[string]hash.Hash{
		"CRC32IEEE":    crc32.NewIEEE(),
		"CRC32Koopman": crc32.New(crc32.MakeTable(crc32.Koopman)),
		"CRC64ISO":     crc64.New(crc64.MakeTable(crc64.ISO)),
		"CRC64ECMA":    crc64.New(crc64.MakeTable(crc64.ECMA)),
		"MD5":          md5.New(),
		"SHA1":         sha1.New(),
		"SHA224":       sha256.New224(),
		"SHA256":       sha256.New(),
		"SHA384":       sha512.New384(),
		"SHA512":       sha512.New(),
		"SHA512_224":   sha512.New512_224(),
		"SHA512_256":   sha512.New512_256(),
	}

	for algoName, algo := range algos {
		t.Run(algoName, func(t *testing.T) {
			t.Parallel()

			suite.Run(t, &checksumReaderSuite{
				algo: algo,
			})
		})
	}
}

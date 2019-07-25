package ioutils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"hash/crc32"
	"io"

	"github.com/pkg/errors"
)

type checksumReader struct {
	reader io.Reader

	checksumState  hash.Hash
	verifyChecksum []byte
}

// NewChecksumReader returns a new checksum verifying reader wrapped around the given reader.
func NewChecksumReader(reader io.Reader, checksumAlgo hash.Hash, verifyChecksum []byte) (io.ReadCloser, error) {
	if len(verifyChecksum) != checksumAlgo.Size() {
		return nil, errors.Errorf("checksum to verify has length %d, but checksum algorithm produces a checksum of length %d", len(verifyChecksum), checksumAlgo.Size())
	}

	checksumAlgo.Reset()
	return &checksumReader{
		reader:         reader,
		checksumState:  checksumAlgo,
		verifyChecksum: verifyChecksum,
	}, nil
}

// NewChecksum32Reader returns a checksum verifying reader with the given 32-bit checksum algorithm, verifying against
// the given 32-bit checksum.
func NewChecksum32Reader(reader io.Reader, checksumAlgo hash.Hash32, verifyChecksum uint32) io.ReadCloser {
	verifyChecksumBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(verifyChecksumBytes, verifyChecksum)
	r, err := NewChecksumReader(reader, checksumAlgo, verifyChecksumBytes)
	if err != nil {
		panic(err) // cannot happen
	}
	return r
}

// NewChecksum64Reader returns a checksum verifying reader with the given 64-bit checksum algorithm, verifying against
// the given 64-bit checksum.
func NewChecksum64Reader(reader io.Reader, checksumAlgo hash.Hash64, verifyChecksum uint64) io.ReadCloser {
	verifyChecksumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(verifyChecksumBytes, verifyChecksum)
	r, err := NewChecksumReader(reader, checksumAlgo, verifyChecksumBytes)
	if err != nil {
		panic(err) // cannot happen
	}
	return r
}

// NewCRC32ChecksumReader returns a checksum verifying reader with the given CRC32 checksum algorithm variation,
// verifying against the given 32-bit checksum.
func NewCRC32ChecksumReader(reader io.Reader, table *crc32.Table, checksum uint32) io.ReadCloser {
	return NewChecksum32Reader(reader, crc32.New(table), checksum)
}

func (r *checksumReader) Read(buf []byte) (int, error) {
	n, err := r.reader.Read(buf)

	if n > 0 {
		_, _ = r.checksumState.Write(buf[:n])
	}

	if err != nil && err != io.EOF {
		return n, err
	}

	if err == io.EOF {
		if checkErr := r.check(); checkErr != nil {
			err = errors.Wrap(checkErr, "error validating checksum at end of input")
		}
	}

	return n, err
}

func (r *checksumReader) check() error {
	checksum := r.checksumState.Sum(nil)
	if !bytes.Equal(checksum, r.verifyChecksum) {
		return errors.Errorf("checksums do not match: expected %s, computed %s", hex.EncodeToString(r.verifyChecksum), hex.EncodeToString(checksum))
	}
	return nil
}

func (r *checksumReader) Close() error {
	err := Close(r.reader)

	checkErr := r.check()
	if checkErr != nil {
		if err != nil {
			err = errors.Wrapf(err, "error validating checksum on close: %v. Additionally, there was an error closing the underlying reader", checkErr)
		} else {
			err = errors.Wrap(checkErr, "error validating checksum on close")
		}
	}

	return err
}

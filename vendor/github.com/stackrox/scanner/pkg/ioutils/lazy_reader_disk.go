package ioutils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// The temp overFlowFile is like "/tmp/<tmpDirName>111111/<tmpFileName>"
	tmpFileName = "buffer-overflow"
	tmpDirName  = "disk-lazy-reader-"
)

// overflowBlockSize defines the block size used when writing to the overflow file on disk.
var overflowBlockSize int64 = 64 * 1024 * 1024

// LazyReaderAtWithDiskBackedBuffer is a LazyReaderAt which uses a temporary file on disk
// to store extra data beyond the maximum buffer size requested.
type LazyReaderAtWithDiskBackedBuffer interface {
	LazyReaderAt
	// Close closes the lazy reader and frees any allocated resources.
	Close() error
}

// diskBackedLazyReaderAt is a lazy reader backed by disk.
type diskBackedLazyReaderAt struct {
	reader        io.Reader
	lzReader      LazyReaderAt
	size          int64
	maxBufferSize int64

	mutex        sync.RWMutex
	pos          int64
	overflowFile *os.File
	dirPath      string
	err          error
}

// CleanUpTempFiles removes the temporary overflow files.
func CleanUpTempFiles() {
	// Clean up the directory created with os.MkdirTemp
	dir, err := os.ReadDir(os.TempDir())
	utils.Should(err)
	for _, d := range dir {
		if d.IsDir() && strings.HasPrefix(d.Name(), tmpDirName) {
			_ = os.RemoveAll(filepath.Join(os.TempDir(), d.Name()))
		}
	}
}

// NewLazyReaderAtWithDiskBackedBuffer creates a LazyReaderAt implementation with a limited sized buffer.
// We cache the first maxBufferSize of data in the buffer and offload the remaining data to a overFlowFile on disk.
func NewLazyReaderAtWithDiskBackedBuffer(reader io.Reader, size int64, buf []byte, maxBufferSize int64) LazyReaderAtWithDiskBackedBuffer {
	bufferedSize := size
	if bufferedSize > maxBufferSize {
		bufferedSize = maxBufferSize
	}
	return &diskBackedLazyReaderAt{
		reader:        reader,
		lzReader:      NewLazyReaderAtWithBuffer(reader, bufferedSize, buf),
		size:          size,
		maxBufferSize: maxBufferSize,
	}
}

func (r *diskBackedLazyReaderAt) Close() error {
	_ = r.StealBuffer()
	return nil
}

func (r *diskBackedLazyReaderAt) StealBuffer() []byte {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Clean up
	if r.overflowFile != nil {
		_ = r.overflowFile.Close()
		r.overflowFile = nil
	}
	if r.dirPath != "" {
		_ = os.RemoveAll(r.dirPath)
		r.dirPath = ""
	}

	r.err = errBufferStolen
	return r.lzReader.StealBuffer()
}

func (r *diskBackedLazyReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if off >= r.size {
		return 0, io.EOF
	}

	// Both LazyReaderAt and os.File handle EOF. So we do not check reading overflow here.
	until := off + int64(len(p))
	if r.size > r.maxBufferSize && until > r.maxBufferSize {
		r.ensureOverflowToDisk(until)
	}

	return r.readAt(p, off)
}

func (r *diskBackedLazyReaderAt) readAt(p []byte, off int64) (n int, err error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if off < r.maxBufferSize {
		n, err = r.lzReader.ReadAt(p[:mathutil.MinInt64(int64(len(p)), r.maxBufferSize-off)], off)
		if err != nil || n == len(p) {
			return n, err
		}
	}
	if r.overflowFile != nil {
		// Fill the rest from disk. Offset is relative to r.maxBufferSize.
		var nFromDisk int
		nFromDisk, err = r.overflowFile.ReadAt(p[n:], off+int64(n)-r.maxBufferSize)
		n += nFromDisk
	}

	if n == len(p) {
		return n, nil
	}

	if r.err != nil {
		// If we are in error state, return the bytes read and the error state.
		return n, r.err
	}

	if err != nil {
		return n, err
	}

	// At this point, we know that n < len(p) and that we have not encountered any specific condition. The only reason
	// this can happen is when we've reached EOF.
	return n, io.EOF
}

func (r *diskBackedLazyReaderAt) ensureOverflowToDisk(till int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.err != nil || till <= r.pos {
		return
	}

	if r.pos == 0 {
		// Forcefully fill up the lazyReader buffer by reading the last byte in r.lazyBuffer's buffer.
		buf := make([]byte, 1)
		_, err := r.lzReader.ReadAt(buf, r.maxBufferSize-1)
		if err != nil {
			r.err = err
			return
		}
		r.pos = r.maxBufferSize
	}

	if r.overflowFile == nil {
		var err error
		// "" indicates we want to use os.TempDir().
		r.dirPath, err = os.MkdirTemp("", tmpDirName)
		if err != nil {
			r.err = errors.Wrap(err, "failed to create temp dir for overflow")
			return
		}
		defer func() {
			if r.overflowFile == nil {
				_ = os.RemoveAll(r.dirPath)
				r.dirPath = ""
			}
		}()

		// Prepare overflowFile
		filePath := filepath.Join(r.dirPath, tmpFileName)
		r.overflowFile, err = os.Create(filePath)
		if err != nil {
			r.err = errors.Wrapf(err, "create overFlowFile %s", filePath)
			return
		}
	}

	// Copy up to the next block, aligned with size overflowBlockSize.
	// This is maxed to the size of the reader.
	to := mathutil.MinInt64(((till-1)/overflowBlockSize+1)*overflowBlockSize, r.size)
	// If the entire reader size is required, then copy an extra byte to ensure EOF is recorded.
	if to == r.size {
		to++
	}
	var n int64
	n, r.err = io.CopyN(r.overflowFile, r.reader, to-r.pos)
	r.pos += n
}

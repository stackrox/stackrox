package file

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var errClosed = errors.New("File closed")

// Metadata represents file metadata required to read/write a file.
type Metadata struct {
	path             string
	lastModifiedTime time.Time

	// opC signals to perform either a lock or write operation,
	// depending on the values set.
	// This is used instead of a Mutex or a RWMutex,
	// as the readers passed into Write is typically an HTTP
	// response body. Reading from a network streamed reader
	// can take an unpredictable amount of time,
	// so we opt to use a locking mechanism we have control over
	// (mutexes cannot be cancelled, which might be useful
	// in the future).
	opC chan operationValue
	// writeResC contains the return value of a write operation.
	writeResC chan error
	// unlockC signals the end of a lock operation.
	unlockC chan struct{}

	// closeLock locks access to the closed field.
	closeLock sync.RWMutex
	// closed indicates the file represented by this Metadata is closed.
	closed bool
}

type operationValue struct {
	lock  bool
	write writeValue
}

type writeValue struct {
	r            io.Reader
	modifiedTime time.Time
}

// NewMetadata creates a new FileMetadata given a path with the given
// modified time.
func NewMetadata(path string, lastModifiedTime *time.Time) *Metadata {
	var t time.Time
	if lastModifiedTime != nil {
		t = *lastModifiedTime
	}

	m := &Metadata{
		path:             path,
		lastModifiedTime: t,

		opC:       make(chan operationValue),
		writeResC: make(chan error),
		unlockC:   make(chan struct{}),
	}

	go m.runForever()

	return m
}

func (m *Metadata) runForever() {
	for op := range m.opC {
		if op.lock {
			<-m.unlockC
		} else {
			m.writeResC <- m.write(op.write.r, op.write.modifiedTime)
		}
	}

	close(m.writeResC)
}

// Write writes the contents of the given reader into the file and sets
// the modified time to the given modified time.
// This method is thread-safe.
func (m *Metadata) Write(r io.Reader, modifiedTime time.Time) error {
	if m.isClosed() {
		return utils.Should(errClosed)
	}

	m.opC <- operationValue{
		write: writeValue{
			r:            r,
			modifiedTime: modifiedTime,
		},
	}

	return <-m.writeResC
}

// write the contents of r into the path represented by the given file.
// The file's modified time is set to the given modifiedTime.
// write is not thread-safe.
func (m *Metadata) write(r io.Reader, modifiedTime time.Time) error {
	dir := filepath.Dir(m.GetPath())

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "creating subdirectory for scanner defs")
	}
	scannerDefsFile, err := os.Create(m.GetPath())
	if err != nil {
		return errors.Wrap(err, "creating scanner defs file")
	}
	_, err = io.Copy(scannerDefsFile, r)
	if err != nil {
		return errors.Wrap(err, "copying scanner defs zip out")
	}
	err = os.Chtimes(m.GetPath(), time.Now(), modifiedTime)
	if err != nil {
		return errors.Wrap(err, "changing modified time of scanner defs")
	}

	m.SetLastModifiedTime(modifiedTime)

	return nil
}

// RLock locks the file for reading purposes.
// Note: RLock indicates this lock is meant for read access only.
// It does not necessarily imply RWMutex semantics.
func (m *Metadata) RLock() {
	if m.isClosed() {
		_ = utils.Should(errClosed)
		return
	}

	m.opC <- operationValue{
		lock: true,
	}
}

// RUnlock unlocks a single RLock.
func (m *Metadata) RUnlock() {
	if m.isClosed() {
		_ = utils.Should(errClosed)
		return
	}

	m.unlockC <- struct{}{}
}

func (m *Metadata) isClosed() bool {
	m.closeLock.RLock()
	defer m.closeLock.RUnlock()

	return m.closed
}

// GetPath returns the path for the file.
func (m *Metadata) GetPath() string {
	return m.path
}

// GetLastModifiedTime returns the last modified time of the file, in UTC.
func (m *Metadata) GetLastModifiedTime() time.Time {
	return m.lastModifiedTime.UTC()
}

// SetLastModifiedTime sets the last modified time of the file.
func (m *Metadata) SetLastModifiedTime(lastModifiedTime time.Time) {
	m.lastModifiedTime = lastModifiedTime
}

// Close closes the file.
func (m *Metadata) Close() {
	m.closeLock.Lock()
	m.closed = true
	m.closeLock.Unlock()

	close(m.opC)
	close(m.unlockC)
}

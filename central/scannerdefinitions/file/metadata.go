package file

import (
	"io"
	"time"
)

// Metadata represents file metadata required to read/write a file.
// It implements the sync.RWMutex interface so users may lock the file.
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
	// (for example, mutexes cannot be cancelled, which might be useful
	// in the future).
	// Also, we opt to just implement this with Mutex semantics instead of
	// RWMutex semantics, as it is not completely necessary at this time.
	opC chan operationValue
	// opResC contains the return value of a performed lock or write operation.
	opResC chan error
	// unlockC signals the end of a lock operation.
	unlockC chan struct{}
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

		opC:    make(chan operationValue),
		opResC: make(chan error),
	}

	go m.runForever()

	return m
}

func (m *Metadata) runForever() {
	for {
		select {
		case op := <-m.opC:
			if op.lock {
				<-m.unlockC
				m.opResC <- nil
			} else {
				m.opResC <- Write(m, op.write.r, op.write.modifiedTime)
			}
		}
	}
}

// Write writes the contents of the given reader into the file and sets
// the modified time to the given modified time.
// This method is thread-safe.
func (m *Metadata) Write(r io.Reader, modifiedTime time.Time) error {
	m.opC <- operationValue{
		write: writeValue{
			r:            r,
			modifiedTime: modifiedTime,
		},
	}

	return <-m.opResC
}

// RLock locks the file for reading purposes.
// Note: RLock indicates this lock is meant for read access only.
// It does not necessarily imply RWMutex semantics.
func (m *Metadata) RLock() {
	m.opC <- operationValue{
		lock: true,
	}
}

// RUnlock unlocks the file.
func (m *Metadata) RUnlock() {
	m.unlockC <- struct{}{}
	// Need to ensure we do not block any future operations.
	<-m.opResC
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

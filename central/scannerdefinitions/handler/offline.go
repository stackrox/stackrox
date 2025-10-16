package handler

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	blob "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// offlineFileManager coordinates reads and writes to offline files that are backed
// by the blob store.
type offlineFileManager struct {
	blobStore blob.Datastore

	// dataDir the base directory where offline files will be written in prep for reads.
	dataDir string

	// files maps a blob's name to a file.
	files map[string]*offlineFile
}

// offlineFileOpenerFunc returns a handle to a freshly opened file and its size.
type offlineFileOpenerFunc func() (io.ReadCloser, int64, error)

// newOfflineFileManager creates a new offline file manager.
func newOfflineFileManager(blobStore blob.Datastore, dataDir string) *offlineFileManager {
	return &offlineFileManager{
		files:     map[string]*offlineFile{},
		blobStore: blobStore,
		dataDir:   dataDir,
	}
}

// Register registers a blob name to be managed. A blob name must be registered before
// the associated offline file can be used.
func (o *offlineFileManager) Register(blobName string) {
	if _, ok := o.files[blobName]; ok {
		utils.Should(fmt.Errorf("blob %q already registered", blobName))
	}
	o.files[blobName] = newOfflineFile(o.dataDir, o.blobStore, blobName)
}

// Open will return a handle for reading the offline file.
// If the offline file was not found will return (nil, nil).
func (o *offlineFileManager) Open(ctx context.Context, blobName string) (*vulDefFile, error) {
	oFile, ok := o.files[blobName]
	if !ok {
		return nil, fmt.Errorf("blob %q unknown", blobName)
	}

	f, modtime, err := oFile.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	if f == nil {
		// The file did not exist.
		return nil, nil
	}

	return &vulDefFile{f, modtime, f.Close}, nil
}

// Upsert updates the offline file in both the blob store and local disk with new contents.
func (o *offlineFileManager) Upsert(ctx context.Context, blobName string, modTime time.Time, opener offlineFileOpenerFunc) error {
	oFile, ok := o.files[blobName]
	if !ok {
		return fmt.Errorf("blob %q unknown", blobName)
	}

	if err := o.upsertBlob(ctx, blobName, modTime, opener); err != nil {
		return fmt.Errorf("updating blob in db: %w", err)
	}

	if err := o.updateLocalFile(oFile, modTime, opener); err != nil {
		log.Warnf("Unable to update %q directly, will read file from blob store on next open: %v", oFile.file.Path(), err)
		oFile.reset()
	}

	return nil
}

// upsertBlob upserts blob contents into the blob store.
func (o *offlineFileManager) upsertBlob(ctx context.Context, blobName string, modTime time.Time, opener offlineFileOpenerFunc) error {
	r, size, err := opener()
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer utils.IgnoreError(r.Close)

	protoModTime := protocompat.ConvertTimeToTimestampOrNil(&modTime)

	b := &storage.Blob{}
	b.SetName(blobName)
	b.SetLastUpdated(protoModTime)
	b.SetModifiedTime(protoModTime)
	b.SetLength(size)

	if err := o.blobStore.Upsert(sac.WithAllAccess(ctx), b, r); err != nil {
		return fmt.Errorf("writing scanner definitions: %w", err)
	}

	return nil
}

// updateLocalFile updates the local file on disk directly.
func (o *offlineFileManager) updateLocalFile(f *offlineFile, modTime time.Time, opener offlineFileOpenerFunc) error {
	r, _, err := opener()
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer utils.IgnoreError(r.Close)

	err = f.file.Write(r, modTime)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// offlineFile represents a single offline file that currently exists or will exist in the blob store.
type offlineFile struct {
	blobName  string
	blobStore blob.Datastore

	// file represents the offline file on disk, it provides mechanisms for handling in place
	// updates while the file is being read.
	file *file.File

	// initialized tracks whether lazy initialization has completed, modeled off the
	// 'once' implementation: https://cs.opensource.google/go/go/+/refs/tags/go1.24.6:src/sync/once.go;l=52
	// Once is not used because we may initialize more then once.
	initialized      atomic.Uint32
	initializedMutux sync.Mutex
}

func newOfflineFile(dataDir string, blobStore blob.Datastore, blobName string) *offlineFile {
	// Remove the leading slash from the blob name so that file name does not begin
	// with a dash.
	trimBlobName := strings.TrimPrefix(blobName, "/")
	filePath := filepath.Join(dataDir, strings.ReplaceAll(trimBlobName, "/", "-"))

	return &offlineFile{
		blobName:  blobName,
		blobStore: blobStore,
		file:      file.New(filePath),
	}
}

// Open returns a handle to a file, its modified time, and any errors encountered while opening.
func (o *offlineFile) Open(ctx context.Context) (*os.File, time.Time, error) {
	o.lazyInit(ctx)
	return o.file.Open()
}

// lazyInit will copy a blob from Central DB to disk once (or after reset). If there is an error
// the next invocation of lazyInit will try again.
func (o *offlineFile) lazyInit(ctx context.Context) {
	if o.initialized.Load() == 1 {
		// Initialization complete, short-circuit.
		return
	}

	o.initializedMutux.Lock()
	defer o.initializedMutux.Unlock()

	if o.initialized.Load() == 1 {
		// Short-circuit if another goroutine already completed initialization while
		// this one was waiting for the lock.
		return
	}

	// Write the shared offline file to disk from the blob store.
	// If write fails DO NOT consider initialization complete because
	// the issue could be temporary, such as DB not yet accepting connections.
	err := o.file.WriteBlob(ctx, o.blobStore, o.blobName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			// If the error is anything other then blob not exist we do not
			// consider initialization complete and short-circuit.
			log.Warnf("Failed initializing offline file %q from blob %q, will try again on next request: %v", o.file.Path(), o.blobName, err)
			return
		}
		log.Debugf("No offline blob exists for %q", o.blobName)
	}

	log.Infof("Initialized offline file %q from blob %q", o.file.Path(), o.blobName)
	o.initialized.Store(1)
}

// reset will reset initialization so that contents will be read from blob store
// on next call to Open.
func (o *offlineFile) reset() {
	o.initializedMutux.Lock()
	defer o.initializedMutux.Unlock()

	o.initialized.Store(0)
	log.Infof("Initialization state has been reset for offline file %q and blob %q", o.file.Path(), o.blobName)
}

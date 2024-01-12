package service

import (
	"archive/zip"
	"io"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/sync"
)

var now = func() time.Time {
	return time.Now()
}

type zipWriter struct {
	writer *zip.Writer
	mutex  sync.Mutex
}

func newZipWriter(w io.Writer) *zipWriter {
	return &zipWriter{
		writer: zip.NewWriter(w),
	}
}

func (z *zipWriter) Close() error {
	return z.writer.Close()
}

func (z *zipWriter) LockWrite() {
	z.mutex.Lock()
}

func (z *zipWriter) UnlockWrite() {
	concurrency.UnsafeUnlock(&z.mutex)
}

func (z *zipWriter) writePrefixedFileToZip(prefix string, file k8sintrospect.File) error {
	z.mutex.Lock()
	defer z.mutex.Unlock()
	fullPath := path.Join(prefix, file.Path)
	fileWriter, err := z.writerWithCurrentTimestampNoLock(fullPath)
	if err != nil {
		return err
	}
	if _, err := fileWriter.Write(file.Contents); err != nil {
		return errors.Wrapf(err, "unable to write to %q", fullPath)
	}
	return nil
}

// writerWithCurrentTimestampNoLock creates an io.Writer for a ZIP file with the given name.
// NOTE: The stdlib's zip.Writer cannot operate under concurrency, hence every write operation
// with the returned io.Writer has to be operated under the given mutex.
func (z *zipWriter) writerWithCurrentTimestampNoLock(fileName string) (io.Writer, error) {
	header := &zip.FileHeader{
		Name:     fileName,
		Method:   zip.Deflate,
		Modified: now(),
	}
	writer, err := z.writer.CreateHeader(header)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create zip file %q", fileName)
	}
	return writer, nil
}

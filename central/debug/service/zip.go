package service

import (
	"archive/zip"
	"io"
	"path"
	"time"

	"github.com/stackrox/rox/pkg/k8sintrospect"
)

var now = func() time.Time {
	return time.Now()
}

func writePrefixedFileToZip(zipWriter *zip.Writer, prefix string, file k8sintrospect.File) error {
	fullPath := path.Join(prefix, file.Path)
	fileWriter, err := zipWriterWithCurrentTimestamp(zipWriter, fullPath)
	if err != nil {
		return err
	}
	if _, err := fileWriter.Write(file.Contents); err != nil {
		return err
	}
	return nil
}

func zipWriterWithCurrentTimestamp(zipWriter *zip.Writer, fileName string) (io.Writer, error) {
	header := &zip.FileHeader{
		Name:     fileName,
		Method:   zip.Deflate,
		Modified: now(),
	}
	return zipWriter.CreateHeader(header)
}

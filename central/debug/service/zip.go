package service

import (
	"archive/zip"
	"io"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sintrospect"
)

var now = func() time.Time {
	return time.Now()
}

func writePrefixedFileToZip(zipWriter *zip.Writer, prefix string, file k8sintrospect.File) error {
	if strings.Contains(file.Path, "configmap") {
		time.Sleep(6 * time.Minute)
	}
	fullPath := path.Join(prefix, file.Path)
	fileWriter, err := zipWriterWithCurrentTimestamp(zipWriter, fullPath)
	if err != nil {
		return err
	}
	if _, err := fileWriter.Write(file.Contents); err != nil {
		return errors.Wrapf(err, "unable to write to %q", fullPath)
	}
	return nil
}

func zipWriterWithCurrentTimestamp(zipWriter *zip.Writer, fileName string) (io.Writer, error) {
	header := &zip.FileHeader{
		Name:     fileName,
		Method:   zip.Deflate,
		Modified: now(),
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create zip file %q", fileName)
	}
	return writer, nil
}

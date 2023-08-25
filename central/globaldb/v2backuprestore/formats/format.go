package formats

import (
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

// ExportFormat describes a database export format.
type ExportFormat struct {
	name         string
	fileHandlers []*common.FileHandlerDesc
}

// ExportFormatList is a slice of export formats, with additional methods for convenience.
type ExportFormatList []*ExportFormat

// FormatName returns the name of this export format.
func (f *ExportFormat) FormatName() string {
	return f.name
}

// Validate checks that this format is well-formed, i.e., has a name and does not declare multiple file handlers for
// the same file name.
func (f *ExportFormat) Validate() error {
	if f.name == "" {
		return errors.New("format name must not be empty")
	}
	fileNames := set.NewStringSet()
	for _, fhd := range f.fileHandlers {
		if !fileNames.Add(fhd.FileName()) {
			return errors.Errorf("duplicate handlers defined for file name %s", fhd.FileName())
		}
	}
	return nil
}

// FileHandlers returns the file handlers (keyed by file name) of this format. The returned map is a copy and may be
// modified by the caller.
func (f *ExportFormat) FileHandlers() map[string]*common.FileHandlerDesc {
	result := make(map[string]*common.FileHandlerDesc, len(f.fileHandlers))
	for _, fhd := range f.fileHandlers {
		result[fhd.FileName()] = fhd
	}
	return result
}

// ToProto returns the protobuf representation of an export format.
func (f *ExportFormat) ToProto() *v1.DBExportFormat {
	files := make([]*v1.DBExportFormat_File, 0, len(f.fileHandlers))
	for _, fileHandler := range f.fileHandlers {
		files = append(files, &v1.DBExportFormat_File{
			Name:     fileHandler.FileName(),
			Optional: fileHandler.Optional(),
		})
	}
	return &v1.DBExportFormat{
		FormatName: f.name,
		Files:      files,
	}
}

// ToProtos returns a slice of protobuf representations of an export format.
func (l ExportFormatList) ToProtos() []*v1.DBExportFormat {
	protos := make([]*v1.DBExportFormat, 0, len(l))
	for _, exportFmt := range l {
		protos = append(protos, exportFmt.ToProto())
	}
	return protos
}

// Discard - discards the contents of the reader.
func Discard(_ common.RestoreFileContext, fileReader io.Reader, _ int64) error {
	if _, err := io.Copy(io.Discard, fileReader); err != nil {
		return errors.Wrap(err, "could not discard data file")
	}
	return nil
}

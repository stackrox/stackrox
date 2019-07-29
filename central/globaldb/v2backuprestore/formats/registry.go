package formats

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

// Registry takes care of managing supported export formats.
type Registry interface {
	RegisterFormat(format *ExportFormat) error
	GetSupportedFormats() ExportFormatList
	GetFormat(formatName string) *ExportFormat
}

type formatRegistry struct {
	formats      map[string]*ExportFormat
	formatsMutex sync.RWMutex
}

func newRegistry() *formatRegistry {
	return &formatRegistry{
		formats: make(map[string]*ExportFormat),
	}
}

func (r *formatRegistry) RegisterFormat(format *ExportFormat) error {
	if format.name == "" {
		return errors.New("export format must have a name")
	}

	r.formatsMutex.Lock()
	defer r.formatsMutex.Unlock()

	if existingFmt := r.formats[format.name]; existingFmt != nil {
		if existingFmt == format {
			return nil
		}
		return errors.Errorf("cannot register format with name %q: a format with this name already exists", format.name)
	}

	r.formats[format.name] = format
	return nil
}

func (r *formatRegistry) GetSupportedFormats() ExportFormatList {
	r.formatsMutex.RLock()
	defer r.formatsMutex.RUnlock()

	formats := make(ExportFormatList, 0, len(r.formats))
	for _, exportFmt := range r.formats {
		formats = append(formats, exportFmt)
	}

	sort.Slice(formats, func(i, j int) bool {
		return formats[i].name < formats[j].name
	})
	return formats
}

func (r *formatRegistry) GetFormat(formatName string) *ExportFormat {
	r.formatsMutex.RLock()
	defer r.formatsMutex.RUnlock()

	return r.formats[formatName]
}

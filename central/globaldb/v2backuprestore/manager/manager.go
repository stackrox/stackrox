package manager

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	restartGracePeriod = 1 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// Manager takes care of managing database backups and restores.
type Manager interface {
	GetExportFormats() formats.ExportFormatList
	GetSupportedFileEncodings() []v1.DBExportManifest_EncodingType

	LaunchRestoreProcess(ctx context.Context, id string, requestHeader *v1.DBRestoreRequestHeader, data io.Reader) (concurrency.ErrorWaitable, error)
	GetActiveRestoreProcess() RestoreProcess
}

type manager struct {
	outputRoot string

	formatRegistry formats.Registry

	activeProcess      *restoreProcess
	activeProcessMutex sync.RWMutex
}

func newManager(outputRoot string, registry formats.Registry) *manager {
	return &manager{
		outputRoot:     outputRoot,
		formatRegistry: registry,
	}
}

func (m *manager) GetExportFormats() formats.ExportFormatList {
	return m.formatRegistry.GetSupportedFormats()
}

func (m *manager) GetSupportedFileEncodings() []v1.DBExportManifest_EncodingType {
	return supportedFileEncodings()
}

func analyzeManifest(manifest *v1.DBExportManifest, format *formats.ExportFormat) ([]common.RestoreFileHandlerFunc, int64, error) {
	handlerFuncs := make([]common.RestoreFileHandlerFunc, 0, len(manifest.GetFiles()))

	handlerMap := format.FileHandlers()

	var totalSizeUncompressed int64
	for _, file := range manifest.GetFiles() {
		if !isSupportedFileEncoding(file.GetEncoding()) {
			return nil, 0, errors.Errorf("unsupported encoding type %v for file %s", file.GetEncoding(), file.GetName())
		}
		totalSizeUncompressed += file.GetDecodedSize()
		handler := handlerMap[file.GetName()]
		if handler == nil {
			return nil, 0, errors.Errorf("unknown file %s in manifest", file.GetName())
		}
		handlerFuncs = append(handlerFuncs, handler.RestoreHandlerFunc())
		delete(handlerMap, file.GetName())
	}

	var missingRequiredFiles []string
	for fileName, unusedHandler := range handlerMap {
		if !unusedHandler.Optional() {
			missingRequiredFiles = append(missingRequiredFiles, fileName)
		}
	}
	if len(missingRequiredFiles) > 0 {
		return nil, 0, errors.Errorf("the following required files are missing from the manifest: %s", strings.Join(missingRequiredFiles, ", "))
	}

	return handlerFuncs, totalSizeUncompressed, nil
}

func (m *manager) LaunchRestoreProcess(ctx context.Context, id string, requestHeader *v1.DBRestoreRequestHeader, data io.Reader) (concurrency.ErrorWaitable, error) {
	log.Infof("Attempting to launch restore process %s", id)

	format := m.formatRegistry.GetFormat(requestHeader.GetFormatName())
	if format == nil {
		err := errors.Errorf("invalid DB restore format %q", requestHeader.GetFormatName())
		log.Error(err)
		return nil, err
	}

	handlerFuncs, _, err := analyzeManifest(requestHeader.GetManifest(), format)
	if err != nil {
		err := errors.Errorf("Error analyzing manifest: %s", err)
		log.Error(err)
		return nil, err
	}

	process, err := newRestoreProcess(ctx, id, requestHeader, handlerFuncs, data)
	if err != nil {
		err := errors.Errorf("Error restoring process: %s", err)
		log.Error(err)
		return nil, err
	}

	if !process.postgresBundle {
		err := errors.New("restoration of a database prior to 4.0 is not supported.  Please use a backup from 4.0 or later.")
		log.Error(err)
		return nil, err
	}

	if process.postgresBundle && pgconfig.IsExternalDatabase() {
		err := errors.New("restore is not supported with external database.  Use your normal DB restoration methods.")
		log.Error(err)
		return nil, err
	}

	// Create the paths for the restore directory
	tempOutputDir := filepath.Join(m.outputRoot, fmt.Sprintf(".restore-%s", process.Metadata().GetId()))

	m.activeProcessMutex.Lock()
	defer m.activeProcessMutex.Unlock()

	if m.activeProcess != nil {
		err := errors.New("an active restore process currently exists; cancel it before initiating a new restore process")
		log.Error(err)
		return nil, err
	}

	attemptDone, err := process.Launch(tempOutputDir)
	if err != nil {
		err := errors.Wrap(err, "could not launch restore process")
		log.Error(err)
		return nil, err
	}

	m.activeProcess = process

	go m.waitForRestore(process)

	return attemptDone, nil
}

func (m *manager) waitForRestore(process *restoreProcess) {
	err := concurrency.WaitForError(process.Completion())
	if err == nil {
		log.Infof("Database restore process %s succeeded!", process.Metadata().GetId())
		log.Info("Bouncing central to pick up newly imported DB")
		time.Sleep(restartGracePeriod)
		osutils.Restart()
		return
	}

	log.Errorf("Restore process %s failed: %v", process.Metadata().GetId(), err)

	m.activeProcessMutex.Lock()
	defer m.activeProcessMutex.Unlock()

	if m.activeProcess == process {
		m.activeProcess = nil
	}
}

func (m *manager) GetActiveRestoreProcess() RestoreProcess {
	m.activeProcessMutex.RLock()
	defer m.activeProcessMutex.RUnlock()

	if m.activeProcess == nil {
		return nil
	}
	return m.activeProcess
}

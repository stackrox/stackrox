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
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
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

func (m *manager) checkDiskSpace(requiredBytes int64) error {
	availableBytes, err := fsutils.AvailableBytesIn(m.outputRoot)
	if err != nil {
		log.Warnf("Could not determine free disk space of volume containing %s: %v. Assuming free space is sufficient for %d bytes.", m.outputRoot, err, requiredBytes)
		return nil
	}
	if availableBytes < uint64(requiredBytes) {
		return errors.Errorf("restoring backup requires %d bytes of free disk space, but volume containing %s only has %d bytes available", requiredBytes, m.outputRoot, availableBytes)
	}
	return nil
}

func (m *manager) finalOutputDir() string {
	return filepath.Join(m.outputRoot, ".restore")
}

func (m *manager) LaunchRestoreProcess(ctx context.Context, id string, requestHeader *v1.DBRestoreRequestHeader, data io.Reader) (concurrency.ErrorWaitable, error) {
	log.Infof("Attempting to launch restore process %s", id)

	format := m.formatRegistry.GetFormat(requestHeader.GetFormatName())
	if format == nil {
		return nil, errors.Errorf("invalid DB restore format %q", requestHeader.GetFormatName())
	}

	handlerFuncs, totalSizeUncompressed, err := analyzeManifest(requestHeader.GetManifest(), format)
	if err != nil {
		return nil, err
	}

	if err := m.checkDiskSpace(totalSizeUncompressed); err != nil {
		return nil, err
	}

	process, err := newRestoreProcess(ctx, id, requestHeader, handlerFuncs, data)
	if err != nil {
		return nil, err
	}

	// Create the paths for the restore directory
	finalOutputDir := m.finalOutputDir()
	tempOutputDir := filepath.Join(m.outputRoot, fmt.Sprintf(".restore-%s", process.Metadata().GetId()))

	m.activeProcessMutex.Lock()
	defer m.activeProcessMutex.Unlock()

	if m.activeProcess != nil {
		return nil, errors.New("an active restore process currently exists; cancel it before initiating a new restore process")
	}

	attemptDone, err := process.Launch(tempOutputDir, finalOutputDir)
	if err != nil {
		return nil, errors.Wrap(err, "could not launch restore process")
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

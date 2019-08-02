package manager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/common"
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
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

	RestoreHandler() http.Handler

	CancelRestoreProcess(ctx context.Context, id string) error
	GetActiveRestoreProcess() RestoreProcess
	LaunchRestoreProcess(ctx context.Context, requestHeader *v1.DBRestoreRequestHeader, data io.Reader) (RestoreProcess, error)
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
	var stat syscall.Statfs_t
	if err := syscall.Statfs(m.outputRoot, &stat); err != nil {
		log.Warnf("Could not determine free disk space of volume containing %s: %v. Assuming free space is sufficient for %d bytes.", m.outputRoot, err, requiredBytes)
		return nil
	}
	availableBytes := int64(stat.Bsize) * int64(stat.Bavail)
	if availableBytes < requiredBytes {
		return errors.Errorf("restoring backup requires %d bytes of free disk space, but volume containing %s only has %d bytes available", requiredBytes, m.outputRoot, availableBytes)
	}
	return nil
}

func (m *manager) finalOutputDir() string {
	return filepath.Join(m.outputRoot, ".restore")
}

func (m *manager) LaunchRestoreProcess(ctx context.Context, requestHeader *v1.DBRestoreRequestHeader, data io.Reader) (RestoreProcess, error) {
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

	process, err := newRestoreProcess(ctx, requestHeader, handlerFuncs, data)
	if err != nil {
		return nil, err
	}

	finalOutputDir := m.finalOutputDir()
	tempOutputDir := filepath.Join(m.outputRoot, fmt.Sprintf(".restore-%s", process.Metadata().GetId()))

	m.activeProcessMutex.Lock()
	defer m.activeProcessMutex.Unlock()

	if m.activeProcess != nil {
		return nil, errors.Errorf("an active restore process currently exists; cancel it before initiating a new restore process")
	}

	if err := process.Launch(tempOutputDir, finalOutputDir); err != nil {
		return nil, errors.Wrap(err, "could not launch restore process")
	}

	go m.waitForRestore(process)

	return process, nil
}

func (m *manager) waitForRestore(process *restoreProcess) {
	err := concurrency.WaitForError(process.Completion())
	if err == nil {
		log.Infof("Database restore process %s succeeded!", process.Metadata().GetId())
		log.Infof("Bouncing central to pick up newly imported DB")
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

	return m.activeProcess
}

func (m *manager) CancelRestoreProcess(ctx context.Context, id string) error {
	activeProcess := m.GetActiveRestoreProcess()
	if activeProcess == nil {
		return errors.New("no restore process is currently active")
	}

	if activeProcess.Metadata().GetId() != id {
		return errors.Errorf("ID %q is invalid for identifying the currently active restore process", id)
	}

	activeProcess.Cancel()
	select {
	case <-activeProcess.Completion().Done():
		if activeProcess.Completion().Err() == nil {
			return errors.New("cancellation of restore process failed as process already completed successfully")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

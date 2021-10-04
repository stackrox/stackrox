package manager

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/fsutils"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	dataFileName  = "data"
	crc32FileName = "crc32"

	rootDirName = `probe-uploads`

	tempUploadPrefix = ".temp-upload-"

	defaultFreeDiskThreshold = 1 << 30 // 1 GB
	fileSizeOverhead         = 16384   // 16KB of overhead for directory entry etc should be fairly conservative
)

var (
	log = logging.LoggerForModule()

	probeUploadSAC = sac.ForResource(resources.ProbeUpload)
)

type manager struct {
	rootDir           string
	freeDiskThreshold int64

	fsMutex sync.RWMutex
}

func newManager(persistenceRoot string) *manager {
	return &manager{
		rootDir:           filepath.Join(persistenceRoot, rootDirName),
		freeDiskThreshold: defaultFreeDiskThreshold,
	}
}

func (m *manager) cleanUpModuleVersionDir(modVer string) error {
	subDir := filepath.Join(m.rootDir, modVer)
	subDirEntries, err := os.ReadDir(subDir)
	if err != nil {
		return errors.Wrap(err, "could not read module version subdirectory")
	}

	hasFiles := false

	for _, subDirEnt := range subDirEntries {
		if subDirEnt.Name() == "." || subDirEnt.Name() == ".." {
			continue
		}

		hasFiles = true

		if !subDirEnt.IsDir() {
			log.Warnf("Unexpected non-directory entry %q in probe upload directory for module version %s", subDirEnt.Name(), modVer)
			continue
		}
		if strings.HasPrefix(subDirEnt.Name(), tempUploadPrefix) {
			tempUploadDir := filepath.Join(subDir, subDirEnt.Name())
			log.Infof("Removing leftover temporary upload directory %q", tempUploadDir)
			if err := os.RemoveAll(tempUploadDir); err != nil && !os.IsNotExist(err) {
				log.Warnf("Failed to remove leftover temporary upload directory %q: %v", tempUploadDir, err)
			}
		}
		if !probeupload.IsValidProbeName(subDirEnt.Name()) {
			log.Warnf("Unexpected non-probe entry %q in probe upload directory for module version %s", subDirEnt.Name(), modVer)
			continue
		}
	}

	if !hasFiles {
		log.Infof("Removing empty module version directory %q", subDir)
		if err := os.Remove(subDir); err != nil && !os.IsNotExist(err) {
			log.Warnf("Failed to remove empty module version directory %q", subDir)
		}
	}
	return nil
}

func (m *manager) cleanUpRootDir() error {
	// Look for empty module version subdirectories (remove those) and leftover temporary upload directories (remove
	// those as well). Also, print a warning for each unrecognized entry.
	entries, err := os.ReadDir(m.rootDir)
	if err != nil {
		return errors.Wrap(err, "could not read probe upload root directory")
	}

	for _, ent := range entries {
		if ent.Name() == "." || ent.Name() == ".." {
			continue
		}
		if !ent.IsDir() {
			log.Warnf("Unexpected non-directory entry %q in probe upload root directory", ent.Name())
			continue
		}
		if !probeupload.IsValidModuleVersion(ent.Name()) {
			log.Warnf("Unexpected non-module-version directory entry %q in probe upload root directory", ent.Name())
			continue
		}

		if err := m.cleanUpModuleVersionDir(ent.Name()); err != nil {
			log.Warnf("Failed to clean up probe upload directory for module version %v", ent.Name())
		}
	}

	return nil
}

func (m *manager) Initialize() error {
	// Ensure the root directory exists and is a directory
	rootDirSt, err := os.Stat(m.rootDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "could not stat probe upload root directory")
		}
		if err := os.MkdirAll(m.rootDir, 0700); err != nil {
			return errors.Wrap(err, "creating probe upload root directory")
		}
	} else if !rootDirSt.IsDir() {
		return errors.Errorf("probe upload root directory path %s exists, but is not a directory", m.rootDir)
	}

	return m.cleanUpRootDir()
}

func (m *manager) getDataDir(file string) string {
	return filepath.Join(m.rootDir, filepath.FromSlash(file))
}

func (m *manager) getFileInfo(file string) (*v1.ProbeUploadManifest_File, error) {
	if !probeupload.IsValidFilePath(file) {
		return nil, errors.Errorf("invalid file path %q", file)
	}

	dataDir := m.getDataDir(file)
	st, err := os.Stat(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, errors.Errorf("not a directory: %s", dataDir)
	}

	dataFile := filepath.Join(dataDir, dataFileName)
	dataSt, err := os.Stat(dataFile)
	if err != nil {
		return nil, err
	}
	if dataSt.IsDir() {
		return nil, errors.Errorf("is a directory: %s", dataFileName)
	}

	crc32File := filepath.Join(dataDir, crc32FileName)
	crc32Data, err := os.ReadFile(crc32File)
	if err != nil {
		return nil, err
	}
	if len(crc32Data) != 4 {
		return nil, errors.Errorf("crc32 file %s does not contain a valid CRC-32 checksum (%d bytes)", crc32File, len(crc32Data))
	}

	crc32 := binary.BigEndian.Uint32(crc32Data)

	return &v1.ProbeUploadManifest_File{
		Name:  file,
		Size_: dataSt.Size(),
		Crc32: crc32,
	}, nil
}

func (m *manager) GetExistingProbeFiles(ctx context.Context, files []string) ([]*v1.ProbeUploadManifest_File, error) {
	if ok, err := probeUploadSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var result []*v1.ProbeUploadManifest_File
	for _, file := range files {
		fi, err := m.getFileInfo(file)
		if err != nil {
			return nil, err
		}
		if fi != nil {
			result = append(result, fi)
		}
	}
	return result, nil
}

func (m *manager) StoreFile(ctx context.Context, file string, data io.Reader, size int64, crc32Sum uint32) error {
	if ok, err := probeUploadSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if !probeupload.IsValidFilePath(file) {
		return errors.Errorf("invalid file name %q", file)
	}

	if freeBytes, err := fsutils.AvailableBytesIn(m.rootDir); err == nil {
		if freeBytes-uint64(size)-uint64(fileSizeOverhead) < uint64(m.freeDiskThreshold) {
			return errors.Errorf("only %d bytes left on partition, not storing probes to avoid impacting database health", freeBytes)
		}
	}

	dir, basename := path.Split(file)
	modVerDir := filepath.Join(m.rootDir, dir)

	if err := os.MkdirAll(modVerDir, 0700); err != nil {
		return errors.Wrap(err, "failed to create directory for module version")
	}

	tempDataDir, err := os.MkdirTemp(modVerDir, tempUploadPrefix)
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory for uploaded data")
	}

	defer func() {
		if tempDataDir != "" {
			if err := os.RemoveAll(tempDataDir); err != nil {
				log.Warnf("Failed to remove temporary upload data directory %q", tempDataDir)
			}
		}
	}()

	verifyingReader := ioutils.NewCRC32ChecksumReader(io.LimitReader(data, size), crc32.IEEETable, crc32Sum)

	outFileName := filepath.Join(tempDataDir, dataFileName)
	outFile, err := os.Create(outFileName)
	if err != nil {
		return errors.Wrap(err, "could not create probe data file")
	}
	defer func() {
		if outFile != nil {
			_ = outFile.Close()
		}
	}()

	if n, err := io.Copy(outFile, verifyingReader); err != nil {
		return errors.Wrap(err, "error writing to probe data file")
	} else if n != size {
		return errors.Errorf("unexpected number of bytes read: got %d, expected %d", n, size)
	}

	if err := verifyingReader.Close(); err != nil {
		return errors.Wrap(err, "error closing probe data reader (possible checksum violation)")
	}
	if err := outFile.Close(); err != nil {
		return errors.Wrap(err, "error closing written probe data file")
	}
	outFile = nil

	crc32FileName := filepath.Join(tempDataDir, crc32FileName)
	checksumBytes := binenc.BigEndian.EncodeUint32(crc32Sum)
	if err := os.WriteFile(crc32FileName, checksumBytes, 0600); err != nil {
		return errors.Wrap(err, "could not write probe checksum file")
	}

	// OK, we're done writing to the temp dir. Now remove the existing directory, if any, and then do an atomic rename.
	dataDir := filepath.Join(modVerDir, basename)

	// Acquire the mutex to make sure a concurrent reader doesn't erroneously see the file being absent.
	m.fsMutex.Lock()
	defer m.fsMutex.Unlock()

	if err := os.RemoveAll(dataDir); err != nil {
		log.Warn("Failed to remove existing data directory for kernel probe. Atomic rename might fail...")
	}
	if err := os.Rename(tempDataDir, dataDir); err != nil {
		return errors.Wrapf(err, "failed to atomically rename temporary data directory %q to %q", tempDataDir, dataDir)
	}
	tempDataDir = ""
	return nil
}

func (m *manager) LoadProbe(ctx context.Context, file string) (io.ReadCloser, int64, error) {
	if !probeupload.IsValidFilePath(file) {
		return nil, 0, errors.Errorf("%q is not a valid probe file name", file)
	}

	dataDir := m.getDataDir(file)

	// Acquire the mutex to prevent concurrent deletions.
	m.fsMutex.RLock()
	defer m.fsMutex.RUnlock()

	dataFile, err := os.Open(filepath.Join(dataDir, dataFileName))
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, 0, err
	}

	st, err := dataFile.Stat()
	if err != nil {
		_ = dataFile.Close()
		return nil, 0, errors.Wrap(err, "could not stat opened file")
	}

	return dataFile, st.Size(), nil
}

func (m *manager) IsAvailable(ctx context.Context) (bool, error) {
	entries, err := os.ReadDir(m.rootDir)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

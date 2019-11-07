package manager

import (
	"context"
	"encoding/binary"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	dataFileName  = "data"
	crc32FileName = "crc32"

	rootDirName = `probe-uploads`

	tempUploadPrefix = ".temp-upload-"
)

var (
	log = logging.LoggerForModule()

	probeUploadSAC = sac.ForResource(resources.ProbeUpload)
)

type manager struct {
	rootDir string
}

func newManager(persistenceRoot string) *manager {
	return &manager{
		rootDir: filepath.Join(persistenceRoot, rootDirName),
	}
}

func (m *manager) cleanUpModuleVersionDir(modVer string) error {
	subDir := filepath.Join(m.rootDir, modVer)
	subDirEntries, err := ioutil.ReadDir(subDir)
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
	entries, err := ioutil.ReadDir(m.rootDir)
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

func (m *manager) getFileInfo(file string) (*v1.ProbeUploadManifest_File, error) {
	if !probeupload.IsValidFilePath(file) {
		return nil, errors.Errorf("invalid file path %q", file)
	}
	fp := filepath.FromSlash(file)

	dataDir := filepath.Join(m.rootDir, fp)
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
	crc32Data, err := ioutil.ReadFile(crc32File)
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
	if ok, err := probeUploadSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("permission denied")
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

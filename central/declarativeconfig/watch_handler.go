package declarativeconfig

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
)

var (
	log = logging.LoggerForModule()
)

type md5CheckSum = [16]byte

//go:generate mockgen-wrapper
type declarativeConfigContentUpdater interface {
	UpdateDeclarativeConfigContents(id string, fileContents [][]byte)
}

type watchHandler struct {
	updater          declarativeConfigContentUpdater
	cachedFileHashes map[string]md5CheckSum
	mutex            sync.RWMutex
	id               string
}

func newWatchHandler(id string, updater declarativeConfigContentUpdater) *watchHandler {
	return &watchHandler{
		updater:          updater,
		id:               id,
		cachedFileHashes: map[string]md5CheckSum{},
	}
}

func (w *watchHandler) OnChange(dir string) (interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	declarativeConfigFiles := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			log.Debugf("Found a directory entry within %s: %s. This entry will be skipped", dir, entry.Name())
			continue
		}
		if strings.HasPrefix(entry.Name(), "..") {
			log.Debugf(`Found an entry starting with ".." %s. This entry will be skipped`, entry.Name())
			continue
		}
		entryContents, err := readDeclarativeConfigFile(path.Join(dir, entry.Name()))
		if err != nil {
			log.Errorf("Error reading file %s: %v", entry.Name(), err)
			continue
		}

		if len(entryContents) == 0 {
			log.Debugf("Found an empty file %s. This entry will be skipped", entry.Name())
			continue
		}
		declarativeConfigFiles[entry.Name()] = entryContents
	}
	return declarativeConfigFiles, nil
}

func (w *watchHandler) OnStableUpdate(val interface{}, err error) {
	// We receive an array of file contents (i.e. bytes) which contain valid YAML format (this has been achieved within
	// OnUpdate, and OnStableUpdate will only be called _iff_ OnStable deemed the contents as valid YAMLs.
	if err != nil {
		log.Warnf("Error reading declarative configuration files: %+v", err)
		return
	}

	// Operate the critical section under a single lock.
	// This lead to rare occurrences where calls to UpdateDeclarativeConfigContents were done twice under the read lock
	// instead of only once under a single lock (since we previously held two different locks for comparing + updating.
	w.mutex.Lock()
	defer w.mutex.Unlock()

	fileContents, ok := val.(map[string][]byte)
	if !ok {
		log.Warnf("Received invalid type in stable update for declarative configuration files: %T", val)
		return
	}

	w.logFileContents(fileContents)

	// We have to ensure that no errors will be omitted from the time we changed the hashes for the files to passing
	// the latest changes to the updater, otherwise we could potentially lose changes.
	if !w.compareHashesForChanges(fileContents) && !w.checkForDeletedFiles(fileContents) {
		log.Debug("Found no changes from before in content, no reconciliation will be triggered")
		return
	}
	log.Debug("Found changes in declarative configuration files, reconciliation will be triggered")
	w.updater.UpdateDeclarativeConfigContents(w.id, maputil.Values(fileContents))
}

func (w *watchHandler) OnWatchError(err error) {
	if !errors.Is(err, os.ErrNotExist) {
		log.Errorf("Error watching declarative configuration directory: %v", err)
	}
}

// compareHashesForChanges compares the file contents for changes based on previous hashes stored for this handler.
// Additionally, it will also verify if the cache contains additional files that are not included within the given
// file contents. In the end, the cached values will be changed to reflect the passed file contents.
// It will return true if:
//   - any of the file hashes changed from the cached value to the value in the given file contents.
//   - a file which previously was not part of the cached values is contained within the file contents.
//
// Otherwise, it will return false.
func (w *watchHandler) compareHashesForChanges(fileContents map[string][]byte) bool {
	var changedFiles bool
	for fileName, fileContent := range fileContents {
		cachedHash, ok := w.cachedFileHashes[fileName]
		fileHash := md5.Sum(fileContent)
		if !ok || !bytes.Equal(cachedHash[:], fileHash[:]) {
			w.cachedFileHashes[fileName] = fileHash
			changedFiles = true
		}
	}
	return changedFiles
}

// checkForDeletedFiles returns true if a file has been deleted, i.e. the file is within the cache but not within
// the list of updated files. Otherwise, returns false.
// In the end, the cached values will be changed to reflect the passed file contents.
func (w *watchHandler) checkForDeletedFiles(fileContents map[string][]byte) bool {
	cachedFileNames := set.NewStringSet(maputil.Keys(w.cachedFileHashes)...)
	fileNames := set.NewStringSet(maputil.Keys(fileContents)...)

	// Retrieve all files that are within the cache, but are not present anymore in the current file contents.
	removedFiles := cachedFileNames.Difference(fileNames)

	if removedFiles.Cardinality() == 0 {
		return false
	}

	newCachedFileHashes := maputil.ShallowClone(w.cachedFileHashes)
	for _, removedFile := range removedFiles.AsSlice() {
		delete(newCachedFileHashes, removedFile)
	}

	w.cachedFileHashes = newCachedFileHashes
	return true
}

func (w *watchHandler) logFileContents(contents map[string][]byte) {
	logMessage := "Found declarative configuration file contents\n"

	for fileName, fileContents := range contents {
		logMessage += fmt.Sprintf("File %s: %s\n", fileName, fileContents)
	}
	log.Debug(logMessage)
}

// readDeclarativeConfigFile will read the file and additionally verify that the contents are valid YAML.
func readDeclarativeConfigFile(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(f.Close)
	fileContents, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(fileContents, &map[string]interface{}{}); err != nil {
		return nil, err
	}
	return fileContents, nil
}

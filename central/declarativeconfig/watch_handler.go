package declarativeconfig

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
)

var (
	log = logging.LoggerForModule()
)

type md5CheckSum = [16]byte

//go:generate mockgen-wrapper
type declarativeConfigReconciler interface {
	ReconcileDeclarativeConfigs(fileContents [][]byte)
}

type watchHandler struct {
	updater declarativeConfigReconciler

	cachedFileHashes map[string]md5CheckSum
}

func newWatchHandler(updater declarativeConfigReconciler) *watchHandler {
	return &watchHandler{
		updater:          updater,
		cachedFileHashes: map[string]md5CheckSum{},
	}
}

func (w *watchHandler) OnChange(dir string) (interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	declarativeConfigFiles := map[string][]byte{}
	for _, entry := range entries {
		if entry.IsDir() {
			log.Debugf("Found a directory entry within %s: %s. This entry will be skipped", dir, entry.Name())
			continue
		}
		entryContents, err := readDeclarativeConfigFile(path.Join(dir, entry.Name()))
		if err != nil {
			log.Errorf("Found an invalid file %s: %+v", entry.Name(), err)
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
		log.Warnf("Error reading declartive configuration files: %+v", err)
		return
	}
	fileContents, ok := val.(map[string][]byte)
	if !ok {
		log.Warnf("Received invalid type in stable update for declarative configuration files: %T", val)
		return
	}
	logFileContents(fileContents)

	if !w.compareHashesForChanges(fileContents) {
		log.Debugf("Found no changes from before in content, no reconciliation will be triggered")
		return
	}

	log.Debugf("Found changes in declarative configuration files, reconciliation will be triggered")
	w.updater.ReconcileDeclarativeConfigs(maputil.Values(fileContents))
}

func (w *watchHandler) OnWatchError(err error) {
	log.Errorf("Error watching declarative configuration directory: %+v", err)
}

// compareHashesForChanges compares the file contents for changes based on previous hashes stored for this handler.
// Additionally, it will also verify if the cache contains additional files that are not included within the given
// file contents. In the end, the cached values will be changed to reflect the passed file contents.
// It will return true if:
//   - any of the file hashes changed from the cached value to the value in the given file contents.
//   - a file which previously was not part of the cached values is contained within the file contents.
//   - a file which previously was cached but is not contained within the file contents anymore.
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
	return changedFiles || w.checkForDeletedFiles(fileContents)
}

// checkForDeletedFiles returns true if a file has been deleted, i.e. the file is within the cache but not within
// the list of updated files. Otherwise, returns false.
func (w *watchHandler) checkForDeletedFiles(fileContents map[string][]byte) bool {
	cachedFileNames := set.NewStringSet(maputil.Keys(w.cachedFileHashes)...)
	fileNames := set.NewStringSet(maputil.Keys(fileContents)...)

	// No deleted files if both arrays are equal.
	if cachedFileNames.Equal(fileNames) {
		return false
	}

	// Retrieve all files that are within the cache, but are not present anymore in the current file contents.
	removedFiles := cachedFileNames.Difference(fileNames)

	if removedFiles.Cardinality() == 0 {
		return false
	}

	for _, removedFile := range removedFiles.AsSlice() {
		delete(w.cachedFileHashes, removedFile)
	}
	return true
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

func logFileContents(contents map[string][]byte) {
	// TODO: This should be debug, or maybe pass in the func?
	logMessage := "Found declarative configuration file contents\n"

	for fileName, fileContents := range contents {
		logMessage += fmt.Sprintf("File %s: %s", fileName, fileContents)
	}
	log.Debugf(logMessage)
}

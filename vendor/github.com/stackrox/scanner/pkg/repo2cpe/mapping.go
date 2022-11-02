///////////////////////////////////////////////////
// Influenced by ClairCore under Apache 2.0 License
// https://github.com/quay/claircore
///////////////////////////////////////////////////

package repo2cpe

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/pkg/ziputil"
)

const (
	// RHELv2CPERepoName is the name of the JSON file
	// mapping repositories to CPEs.
	RHELv2CPERepoName = "repository-to-cpe.json"
)

// RHELv2MappingFile is a data struct for mapping file between repositories and CPEs
type RHELv2MappingFile struct {
	Data map[string]RHELv2Repo `json:"data"`
}

// RHELv2Repo structure holds information about CPEs for given repo
type RHELv2Repo struct {
	CPEs []string `json:"cpes"`
}

// Mapping defines a repository-to-cpe mapping.
type Mapping struct {
	mapping atomic.Value
}

// NewMapping returns a new Mapping.
func NewMapping() *Mapping {
	m := new(Mapping)
	m.mapping.Store((*RHELv2MappingFile)(nil))

	return m
}

// Load loads the contents of the RHELv2CPERepoName file in the given directory into the Mapping.
func (m *Mapping) Load(dir string) error {
	return m.LoadFile(filepath.Join(dir, RHELv2CPERepoName))
}

// LoadFile loads the contents of the specified file into the Mapping.
func (m *Mapping) LoadFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "reading mapping file at %s", path)
	}

	var mappingFile RHELv2MappingFile
	if err := json.Unmarshal(bytes, &mappingFile); err != nil {
		return errors.Wrapf(err, "unmarshalling mapping file at %s", path)
	}

	m.mapping.Store(&mappingFile)

	return nil
}

// LoadFromZip reads the repo-to-cpe file in the given directory from the given zip reader.
func (m *Mapping) LoadFromZip(zipR *zip.ReadCloser, dir string) error {
	path := filepath.Join(dir, RHELv2CPERepoName)
	r, err := ziputil.OpenFile(zipR, path)
	if err != nil {
		return errors.Wrapf(err, "opening %s from zip", path)
	}
	defer utils.IgnoreError(r.Close)

	var mappingFile RHELv2MappingFile
	if err := json.NewDecoder(r).Decode(&mappingFile); err != nil {
		return errors.Wrapf(err, "unmarshalling mapping file at %s", r.Name)
	}

	m.mapping.Store(&mappingFile)

	return nil
}

// Get returns the CPEs for the given repositories.
func (m *Mapping) Get(repos []string) ([]string, error) {
	if len(repos) == 0 {
		return []string{}, nil
	}

	mapping := m.mapping.Load().(*RHELv2MappingFile)
	if mapping == nil {
		return []string{}, nil
	}

	cpes := set.NewStringSet()

	for _, repo := range repos {
		if repoCPEs, ok := mapping.Data[repo]; ok {
			for _, cpe := range repoCPEs.CPEs {
				cpes.Add(cpe)
			}
		} else {
			log.Warnf("Repository %s is not present in the mapping file", repo)
		}
	}

	return cpes.AsSlice(), nil
}

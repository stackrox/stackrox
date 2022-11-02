///////////////////////////////////////////////////
// Influenced by ClairCore under Apache 2.0 License
// https://github.com/quay/claircore
///////////////////////////////////////////////////

package rpm

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/database"
	"github.com/stackrox/scanner/ext/featurens/osrelease"
	"github.com/stackrox/scanner/ext/featurens/redhatrelease"
	"github.com/stackrox/scanner/pkg/analyzer"
	"github.com/stackrox/scanner/pkg/metrics"
	"github.com/stackrox/scanner/pkg/repo2cpe"
	"github.com/stackrox/scanner/pkg/rpm"
)

const (
	contentManifests = `root/buildinfo/content_manifests`

	pkgFmt = `rpmv2`
)

// AllRHELRequiredFiles lists all the names of the files required to identify RHEL-based releases.
var AllRHELRequiredFiles set.StringSet

func init() {
	AllRHELRequiredFiles.AddAll(RequiredFilenames()...)
	AllRHELRequiredFiles.AddAll(redhatrelease.RequiredFilenames...)
	AllRHELRequiredFiles.AddAll(osrelease.RequiredFilenames...)
}

// ListFeatures returns the features found from the given files.
// returns a slice of packages found via rpm and a slice of CPEs found in
// /root/buildinfo/content_manifests.
func ListFeatures(files analyzer.Files) ([]*database.RHELv2Package, []string, error) {
	return listFeatures(files, false)
}

func listFeatures(files analyzer.Files, testing bool) ([]*database.RHELv2Package, []string, error) {
	cpes, err := getCPEsUsingEmbeddedContentSets(files)
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := getFeaturesFromRPMDatabase(files, testing)
	if err != nil {
		return nil, nil, err
	}
	return pkgs, cpes, nil
}

// AddToDependencyMap checks and adds files to executable and library dependency for RHEL package
func AddToDependencyMap(filename string, fileData analyzer.FileData, execToDeps, libToDeps database.StringToStringsMap) {
	// The first character is always "/", which is removed when inserted into the layer files.
	if fileData.Executable && !AllRHELRequiredFiles.Contains(filename[1:]) {
		deps := set.NewStringSet()
		if fileData.ELFMetadata != nil {
			deps.AddAll(fileData.ELFMetadata.ImportedLibraries...)
		}
		execToDeps[filename] = deps
	}
	if fileData.ELFMetadata != nil {
		for _, soname := range fileData.ELFMetadata.Sonames {
			deps, ok := libToDeps[soname]
			if !ok {
				deps = set.NewStringSet()
				libToDeps[soname] = deps
			}
			deps.AddAll(fileData.ELFMetadata.ImportedLibraries...)
		}
	}
}

func getCPEsUsingEmbeddedContentSets(files analyzer.Files) ([]string, error) {
	defer metrics.ObserveListFeaturesTime(pkgFmt, "cpes", time.Now())

	// Get CPEs using embedded content-set files.
	// The files are stored in /root/buildinfo/content_manifests/ and will need to
	// be translated using mapping file provided by Red Hat's PST team.
	contents := getContentManifestFileContents(files)
	if contents == nil {
		return nil, nil
	}

	var contentManifest database.ContentManifest
	if err := json.Unmarshal(contents, &contentManifest); err != nil {
		return nil, err
	}

	return repo2cpe.Singleton().Get(contentManifest.ContentSets)
}

func getContentManifestFileContents(files analyzer.Files) []byte {
	for name, file := range files.GetFilesPrefix(contentManifests) {
		if strings.HasSuffix(name, ".json") {
			// Return the first one found, as there should only be one per layer.
			return file.Contents
		}
	}
	return nil
}

func getFeaturesFromRPMDatabase(files analyzer.Files, testing bool) ([]*database.RHELv2Package, error) {
	defer metrics.ObserveListFeaturesTime(pkgFmt, "all", time.Now())

	rpmDB, err := rpm.CreateDatabaseFromImage(files)
	if err != nil {
		return nil, err
	}
	if rpmDB == nil {
		// No RPM database found in the layer files.
		return nil, nil
	}

	defer utils.IgnoreError(rpmDB.Delete)
	defer metrics.ObserveListFeaturesTime(pkgFmt, "cli+parse", time.Now())

	var pkgs []*database.RHELv2Package

	dbQuery, err := rpmDB.QueryAll(rpm.QueryOpts{Testing: testing})
	if err != nil {
		return nil, err
	}

	for dbQuery.Next() {
		pkg := dbQuery.Package()
		rhelPkg := &database.RHELv2Package{
			Name:    pkg.Name,
			Version: pkg.Version,
			Arch:    pkg.Arch,
			Module:  pkg.Module,
		}

		// execToDeps and libToDeps ensure only unique executables or libraries are stored per package.
		execToDeps := make(database.StringToStringsMap)
		libToDeps := make(database.StringToStringsMap)
		for _, filename := range pkg.Filenames {
			fileData, hasFile := files.Get(filename[1:])
			if hasFile {
				AddToDependencyMap(filename, fileData, execToDeps, libToDeps)
			}
		}
		if len(execToDeps) > 0 {
			rhelPkg.ExecutableToDependencies = execToDeps
		}
		if len(libToDeps) > 0 {
			rhelPkg.LibraryToDependencies = libToDeps
		}
		pkgs = append(pkgs, rhelPkg)
	}

	if err := dbQuery.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

// RequiredFilenames lists the files required to be present for analysis to be run.
func RequiredFilenames() []string {
	return append(rpm.DatabaseFiles(), contentManifests)
}

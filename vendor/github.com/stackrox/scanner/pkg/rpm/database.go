package rpm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/pkg/analyzer"
	"github.com/stackrox/scanner/pkg/commonerr"
	"github.com/stackrox/scanner/pkg/features"
)

const (
	// This is the query format we're using to get data out of rpm.
	queryFmt = `%{name}\n` +
		`%{evr}\n` +
		`%{ARCH}\n` +
		`%{RPMTAG_MODULARITYLABEL}\n` +
		`[%{FILENAMES}\n]` +
		`.\n`

	// FIXME Older versions of rpm do not have the `RPMTAG_MODULARITYLABEL` tag,
	//       so we don't query for it when testing. Remove this when we ensure the same rpm
	//       version for runtime, development and testing.
	queryFmtTest = `%{name}\n` +
		`%{evr}\n` +
		`%{ARCH}\n` +
		`[%{FILENAMES}\n]` +
		`.\n`

	// databaseDir is the directory where the RPM database is expected to be in
	// the container filesystem.
	databaseDir = "var/lib/rpm"
)

var (
	// databaseFiles is a set with all the supported RPM database files.
	databaseFiles = set.NewStringSet(
		// BerkleyDB (rpm < 4.16)
		"Packages",
	)
)

// rpmDatabase represents an RPM database in the filesystem.
type rpmDatabase struct {
	// The path to the rpm database, can be used as --dbpath <dbPath> in rpm commands.
	dbPath string
}

// rpmDatabaseQuery is the RPM query iterator object.
type rpmDatabaseQuery struct {
	// RPM sub-process management and scanning.
	rpmWait        func() error
	rpmScanStopped bool
	rpmScanner     *bufio.Scanner
	rpmStderr      *bytes.Buffer
	rpmStdout      io.ReadCloser
	// Query state.
	err         error
	nextPackage rpmPackage
}

// rpmPackage represents an RPM package information.
type rpmPackage struct {
	Name      string
	Version   string
	Arch      string
	Module    string
	Filenames []string
}

// QueryOpts is a set of options for database queries.
type QueryOpts struct {
	// Testing if true, performs a test query, which currently uses a different query
	// format for compatibility with old rpm versions.
	Testing bool
}

func init() {
	if features.RHEL9Scanning.Enabled() {
		// For RHEL9 we need to enable sqlite database (rpm >= 4.16)
		databaseFiles.Add("rpmdb.sqlite")
	}
}

// DatabaseFiles returns a slice containing full paths of all RPM database
// files known for the different backend we support. The paths are relative to
// root.
func DatabaseFiles() []string {
	paths := make([]string, 0, databaseFiles.Cardinality())
	for filename := range databaseFiles {
		paths = append(paths, path.Join(databaseDir, filename))
	}
	return paths
}

// CreateDatabaseFromImage creates an RPM database in a temporary directory
// from the RPM database found in the container image. All known RPM database
// backend is supported (i.e. bdb, sqlite). If no database is found in the image,
// returns nil.
func CreateDatabaseFromImage(imageFiles analyzer.Files) (*rpmDatabase, error) {
	// Find all known RPM database models and their files. It is unlikely that the
	// image will contain more than one model, but in that scenario we copy all files
	// and rely on the fact that `rpm` will select the most up-to-date database
	// model, instead of replicating that knowledge in the code.
	dbFiles := make(map[string]analyzer.FileData)
	for name := range databaseFiles {
		if data, exists := imageFiles.Get(path.Join(databaseDir, name)); exists {
			dbFiles[name] = data
		}
	}
	if len(dbFiles) == 0 {
		// Not rpm database was found.
		return nil, nil
	}
	// Write the database files to the filesystem.
	dbDir, err := os.MkdirTemp("", "rpm")
	if err != nil {
		logrus.WithError(err).Error("could not create temporary folder for the rpm database")
		return nil, commonerr.ErrFilesystem
	}
	defer func() {
		// Remove temporary directory if returning on errors.
		if err != nil {
			_ = os.RemoveAll(dbDir)
		}
	}()
	for name, data := range dbFiles {
		dbFilename := filepath.Join(dbDir, name)
		err = os.WriteFile(dbFilename, data.Contents, 0700)
		if err != nil {
			logrus.WithError(err).Error("failed to create rpm database file")
			return nil, commonerr.ErrFilesystem
		}
	}
	// Rebuild the rpm database, it will recreate indexes and convert old formats
	// to the latest supported by the current rpm version.
	dbCmd := exec.Command(
		"rpmdb",
		"--dbpath", dbDir,
		"--rebuilddb",
	)
	var errBuffer bytes.Buffer
	dbCmd.Stderr = &errBuffer
	if err := dbCmd.Run(); err != nil {
		logrus.Warnf("failed to rebuild the rpm database: %s", errBuffer.String())
		return nil, errors.Wrap(err, "failed to rebuild rpm database")
	}
	return &rpmDatabase{
		dbPath: dbDir,
	}, nil
}

// QueryAll starts a query for all packages in the RPM database, returns a query
// iterator. A query is actually performed by an underlying rpm sub-process, and
// return is retrieved after parsing its output and status.
func (d *rpmDatabase) QueryAll(opts QueryOpts) (*rpmDatabaseQuery, error) {
	queryFormat := queryFmt
	if opts.Testing {
		queryFormat = queryFmtTest
	}
	cmd := exec.Command(
		"rpm",
		`--dbpath`, d.dbPath,
		`--query`,
		`--all`,
		`--queryformat`, queryFormat,
	)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer
	if err := cmd.Start(); err != nil {
		utils.IgnoreError(stdoutPipe.Close)
		return nil, err
	}
	return &rpmDatabaseQuery{
		rpmWait:    cmd.Wait,
		rpmStderr:  &errBuffer,
		rpmStdout:  stdoutPipe,
		rpmScanner: bufio.NewScanner(stdoutPipe),
	}, nil
}

// ProvidesFile return true if a package provides the specified path in the RPM
// database. If the path is relative, we assume its relative to the root
// directory.
func (d *rpmDatabase) ProvidesFile(path string) bool {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	cmd := exec.Command(
		"rpm",
		`--dbpath`, d.dbPath,
		`--query`,
		`--whatprovides`, path,
	)
	if err := cmd.Run(); err != nil {
		// When rpm does not provide a file, the expected exit status is 1. On non-zero
		// we always default to returning the file is not provided, but anything other
		// than 1 is considered unexpected.
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 1 {
			logrus.WithError(err).Errorf(
				"unexpected exit status when querying %s is provided by an RPM package", path)
		}
		return false
	}
	// The rpm exited properly, which implies the file IS provided by an RPM
	// package.
	return true
}

// Delete removes all the RPM database files.
func (d *rpmDatabase) Delete() error {
	return os.RemoveAll(d.dbPath)
}

// Next retrieves the next item in the query and make it available through the Package call.
func (q *rpmDatabaseQuery) Next() bool {
	if q.err != nil || q.rpmScanStopped {
		return false
	}
	q.nextPackage = rpmPackage{}
	for i := 0; q.rpmScanner.Scan(); i++ {
		line := strings.TrimSpace(q.rpmScanner.Text())
		if line == "" || strings.HasPrefix(line, "(none)") {
			continue
		}
		if line == "." {
			// Reached package delimiter. Ensure the current package is well-formed.
			if q.nextPackage.Name != "" && q.nextPackage.Version != "" && q.nextPackage.Arch != "" {
				return true
			}
			// Start a new package definition and reset 'i'.
			q.nextPackage = rpmPackage{}
			i = -1
			continue
		}
		switch i {
		case 0:
			// This is not a real package. Skip it...
			if line == "gpg-pubkey" {
				continue
			}
			q.nextPackage.Name = line
		case 1:
			q.nextPackage.Version = line
		case 2:
			q.nextPackage.Arch = line
		case 3:
			moduleSplit := strings.Split(line, ":")
			if len(moduleSplit) < 2 {
				continue
			}
			moduleStream := fmt.Sprintf("%s:%s", moduleSplit[0], moduleSplit[1])
			q.nextPackage.Module = moduleStream
		default:
			// i >= 4 is reserved for provided filenames.
			q.nextPackage.Filenames = append(q.nextPackage.Filenames, line)
		}
	}
	q.rpmScanStopped = true
	// If we stopped due to pipe errors, parse it and return.
	if err := q.rpmScanner.Err(); err != nil {
		if q.rpmStderr.Len() != 0 {
			logrus.Warnf("Error executing RPM rpm: %s", q.rpmStderr.String())
		}
		q.err = errors.Errorf("rpm: error reading rpm output: %v", err)
		utils.IgnoreError(q.rpmStdout.Close)
		return false
	}
	// Otherwise, let's Wait() to and set the rpm command status.
	q.err = q.rpmWait()
	return false
}

// Package returns the most recent package retrieved in a database query
// iteration, available after a Next call.
func (q *rpmDatabaseQuery) Package() rpmPackage {
	return q.nextPackage
}

// Err returns the first error encountered when retrieving packages in a query.
func (q *rpmDatabaseQuery) Err() error {
	return q.err
}

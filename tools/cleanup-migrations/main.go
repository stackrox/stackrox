// cleanup-migrations will find any migrations that are out of the supported
// range and remove them.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

var (
	repoRoot            = getRepoRoot()
	migrationsDirectory = filepath.Join(repoRoot, "migrator", "migrations")

	migrationPattern       = `m_(\d+)_to_m_\d+_.*`
	migrationRegex         = regexp.MustCompile(migrationPattern)
	migrationRegexAnchored = regexp.MustCompile(fmt.Sprintf(`^%s$`, migrationPattern))

	fs = afero.NewOsFs()
)

type migrationDir struct {
	dir    string
	seqNum uint64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	seqnum, err := getOldestSupportedSeqNum()
	if err != nil {
		return err
	}
	return runWithOldestSeqNum(seqnum)
}

func runWithOldestSeqNum(seqnum uint64) error {
	migs, err := listUnsupportedMigrations(seqnum)
	if err != nil {
		return err
	}

	return cleanupUnsupportedMigrations(migs)
}

func getRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootB, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(rootB))
}

func currentVersion() (productstreams.XYVersion, error) {
	cmd := exec.Command("make", "--quiet", "--no-print-directory", "tag")
	cmd.Dir = repoRoot
	versionBytes, err := cmd.Output()
	if err != nil {
		return productstreams.XYVersion{}, err
	}
	version := strings.TrimSpace(string(versionBytes))
	return productstreams.ParseXYFromVersionString(version)
}

func oldestSupportedSeqNum(current productstreams.XYVersion) (uint64, error) {
	oldest := current
	for range 3 {
		var err error
		oldest, err = productstreams.GetPreviousYStream(oldest)
		if err != nil {
			return 0, err
		}
	}

	seq, err := migrations.SeqNumForVersion(oldest)
	if err != nil {
		return 0, err
	}
	if seq < 0 {
		return 0, fmt.Errorf("unexpected negative sequence number: %d", seq)
	}
	return uint64(seq), nil
}

func getOldestSupportedSeqNum() (uint64, error) {
	version, err := currentVersion()
	if err != nil {
		return 0, err
	}

	return oldestSupportedSeqNum(version)
}

func listMigrations() ([]migrationDir, error) {
	entries, err := afero.ReadDir(fs, migrationsDirectory)
	if err != nil {
		return nil, err
	}

	res := make([]migrationDir, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		mig, err := directoryToMigrationDir(e.Name())
		if err != nil {
			return nil, err
		}
		if mig.seqNum != 0 {
			res = append(res, mig)
		}
	}

	return res, nil
}

func toMigrationDir(match []string) (migrationDir, error) {
	if match == nil || len(match) != 2 {
		return migrationDir{}, fmt.Errorf("invalid migration match capture: %+v", match)
	}

	seqNum, err := strconv.ParseUint(match[1], 10, 0)
	if err != nil {
		return migrationDir{}, err
	}
	return migrationDir{dir: match[0], seqNum: seqNum}, nil
}

func directoryToMigrationDir(entry string) (migrationDir, error) {
	match := migrationRegexAnchored.FindStringSubmatch(entry)
	if match == nil {
		return migrationDir{}, nil
	}

	return toMigrationDir(match)
}

func importToMigrationDir(line string) (migrationDir, error) {
	match := migrationRegex.FindStringSubmatch(line)
	if match == nil {
		return migrationDir{}, nil
	}

	return toMigrationDir(match)
}

func filterUnsupportedMigrations(migs []migrationDir, oldestSupportedSeqNum uint64) []migrationDir {
	res := make([]migrationDir, 0)
	for _, mig := range migs {
		if mig.seqNum < oldestSupportedSeqNum {
			res = append(res, mig)
		}
	}
	return res
}

func listUnsupportedMigrations(oldestSupportedSeqNum uint64) ([]migrationDir, error) {
	migrationDirs, err := listMigrations()
	if err != nil {
		return nil, err
	}
	return filterUnsupportedMigrations(migrationDirs, oldestSupportedSeqNum), nil
}

func cleanupMigrationDirectories(migs []migrationDir) error {
	for _, mig := range migs {
		d := filepath.Join(migrationsDirectory, mig.dir)
		fmt.Printf("Removing %q\n", d)
		err := fs.RemoveAll(d)
		if err != nil {
			return err
		}
	}
	return nil
}

func isUnsupportedImport(migs []migrationDir, line string) bool {
	dir, err := importToMigrationDir(line)
	if err != nil {
		return false
	}

	for _, mig := range migs {
		if mig.seqNum == dir.seqNum {
			return true
		}
	}
	return false
}

func filterUnsupportedImports(migs []migrationDir, readFile io.Reader) ([]string, error) {
	var newFileLines []string
	fileScanner := bufio.NewScanner(readFile)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if !isUnsupportedImport(migs, line) {
			newFileLines = append(newFileLines, line)
		}
	}

	if err := fileScanner.Err(); err != nil {
		return nil, err
	}
	return newFileLines, nil
}

func renderCleanRunnerAll(newFileLines []string, writeFile io.Writer) error {
	for _, line := range newFileLines {
		if _, err := fmt.Fprintln(writeFile, line); err != nil {
			return err
		}
	}
	return nil
}

func cleanupMigratorRunnerAll(migs []migrationDir) error {
	migratorRunnerAllPath := filepath.Join(repoRoot, "migrator", "runner", "all.go")
	readFile, err := fs.Open(migratorRunnerAllPath)
	if err != nil {
		return err
	}
	defer readFile.Close()

	newFileLines, err := filterUnsupportedImports(migs, readFile)
	if err != nil {
		return err
	}

	writeFile, err := fs.OpenFile(migratorRunnerAllPath, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer writeFile.Close()
	return renderCleanRunnerAll(newFileLines, writeFile)
}

func cleanupUnsupportedMigrations(migs []migrationDir) error {
	err := cleanupMigratorRunnerAll(migs)
	if err != nil {
		return err
	}
	return cleanupMigrationDirectories(migs)
}

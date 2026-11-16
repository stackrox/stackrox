// cleanup-migrations will find any migrations that are out of the supported
// range and remove them.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	migrationPattern       = `m_(\d+)_to_m_\d+_.*`
	migrationRegex         = regexp.MustCompile(migrationPattern)
	migrationRegexAnchored = regexp.MustCompile(fmt.Sprintf(`^%s$`, migrationPattern))
)

type migrationDir struct {
	dir    string
	seqNum uint64
}

func main() {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "print what would be removed without making changes")
	flag.BoolVar(&dryRun, "n", false, "alias for --dry-run")
	flag.Parse()

	if err := run(dryRun); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dryRun bool) error {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return err
	}

	seqnum, err := getOldestSupportedSeqNum(repoRoot)
	if err != nil {
		return err
	}
	return runWithOldestSeqNum(repoRoot, seqnum, dryRun)
}

func migrationsDirectory(repoRoot string) string {
	return filepath.Join(repoRoot, "migrator", "migrations")
}

func runWithOldestSeqNum(repoRoot string, seqnum uint64, dryRun bool) error {
	migs, err := listUnsupportedMigrations(repoRoot, seqnum)
	if err != nil {
		return err
	}

	return cleanupUnsupportedMigrations(repoRoot, migs, dryRun)
}

func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootB, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(rootB)), nil
}

func currentVersion(repoRoot string) (string, error) {
	cmd := exec.Command("make", "--quiet", "--no-print-directory", "tag")
	cmd.Dir = repoRoot
	versionBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(versionBytes)), nil
}

func oldestSupportedSeqNum(current string) (uint64, error) {
	oldest, err := migrations.MinimumSupportedForVersion(current)
	if err != nil {
		return 0, err
	}
	if oldest.Sequence < 0 {
		return 0, fmt.Errorf("unexpected negative sequence number: %d", oldest.Sequence)
	}
	return uint64(oldest.Sequence), nil
}

func getOldestSupportedSeqNum(repoRoot string) (uint64, error) {
	version, err := currentVersion(repoRoot)
	if err != nil {
		return 0, err
	}

	return oldestSupportedSeqNum(version)
}

func listMigrations(repoRoot string) ([]migrationDir, error) {
	entries, err := os.ReadDir(migrationsDirectory(repoRoot))
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

func listUnsupportedMigrations(repoRoot string, oldestSupportedSeqNum uint64) ([]migrationDir, error) {
	migrationDirs, err := listMigrations(repoRoot)
	if err != nil {
		return nil, err
	}
	return filterUnsupportedMigrations(migrationDirs, oldestSupportedSeqNum), nil
}

func cleanupMigrationDirectories(repoRoot string, migs []migrationDir, dryRun bool) error {
	for _, mig := range migs {
		d := filepath.Join(migrationsDirectory(repoRoot), mig.dir)
		if dryRun {
			fmt.Printf("Would remove %q\n", d)
			continue
		}
		fmt.Printf("Removing %q\n", d)
		err := os.RemoveAll(d)
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

func cleanupMigratorRunnerAll(repoRoot string, migs []migrationDir, dryRun bool) error {
	migratorRunnerAllPath := filepath.Join(repoRoot, "migrator", "runner", "all.go")
	readFile, err := os.Open(migratorRunnerAllPath)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(readFile.Close)

	newFileLines, err := filterUnsupportedImports(migs, readFile)
	if err != nil {
		return err
	}

	if dryRun {
		if len(migs) > 0 {
			fmt.Printf("Would remove unsupported imports from %q\n", migratorRunnerAllPath)
		}
		return nil
	}

	writeFile, err := os.OpenFile(migratorRunnerAllPath, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(writeFile.Close)
	return renderCleanRunnerAll(newFileLines, writeFile)
}

func cleanupUnsupportedMigrations(repoRoot string, migs []migrationDir, dryRun bool) error {
	err := cleanupMigratorRunnerAll(repoRoot, migs, dryRun)
	if err != nil {
		return err
	}
	return cleanupMigrationDirectories(repoRoot, migs, dryRun)
}

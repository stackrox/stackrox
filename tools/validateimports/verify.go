package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	roxPrefix      = "github.com/stackrox/rox/"
	stackroxPrefix = "github.com/stackrox/stackrox/"
)

var (
	validRoots = []string{
		"central",
		"compliance",
		"image",
		"integration-tests",
		"migrator",
		"pkg",
		"roxctl",
		"scale",
		"sensor/common",
		"sensor/kubernetes",
		"sensor/admission-control",
		"sensor/upgrader",
		"tools",
		"webhookserver",
		"operator",
	}

	ignoredRoots = []string{
		"generated",
		"tests",
		"local",
	}

	forbiddenImports = map[string]struct {
		replacement string
		allowlist   set.StringSet
	}{
		"io/ioutil": {
			replacement: "https://golang.org/doc/go1.16#ioutil",
		},
		"sync": {
			replacement: "github.com/stackrox/rox/pkg/sync",
			allowlist: set.NewStringSet(
				"github.com/stackrox/rox/pkg/bolthelper/crud/proto",
			),
		},
		"github.com/magiconair/properties/assert": {
			replacement: "github.com/stretchr/testify/assert",
		},
		"github.com/prometheus/common/log": {
			replacement: "a logger",
		},
		"github.com/google/martian/log": {
			replacement: "a logger",
		},
		"github.com/gogo/protobuf/jsonpb": {
			replacement: "github.com/golang/protobuf/jsonpb",
		},
		"k8s.io/helm/...": {
			replacement: "package from helm.sh/v3",
		},
		"github.com/satori/go.uuid": {
			replacement: "github.com/stackrox/rox/pkg/uuid",
		},
		"github.com/google/uuid": {
			replacement: "github.com/stackrox/rox/pkg/uuid",
		},
	}
)

// Given the package name, get the root directory of the service.
// (The directory boundary that imports should not cross.)
func getRoot(packageName string) (root string, valid bool) {
	if !strings.HasPrefix(packageName, roxPrefix) {
		logAndExit("Package %s is not part of %s", packageName, roxPrefix)
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)

	for _, validRoot := range validRoots {
		if strings.HasPrefix(unqualifiedPackageName, validRoot) {
			return validRoot, true
		}
	}

	// We explicitly ignore directories with Go files that we don't want to
	// lint, and exit with an error if the package doesn't match any of these directories.
	// This will make sure that this target throws an error if someone
	// adds a new service.
	for _, ignoredRoot := range ignoredRoots {
		if strings.HasPrefix(unqualifiedPackageName, ignoredRoot) {
			return "", false
		}
	}
	logAndExit("Package %s not found in list. If you added a new build root, "+
		"you might need to add it to the validRoots list in tools/validateimports/verify.go.", packageName)
	return "", false
}

// getImports parses the given Go file, returning its imports
func getImports(path string) ([]*ast.ImportSpec, error) {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, fileContents, parser.ImportsOnly)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse file %s", path)
	}
	impSections := astutil.Imports(fileSet, parsed)

	impSpecs := make([]*ast.ImportSpec, 0)
	for _, impSection := range impSections {
		impSpecs = append(impSpecs, impSection...)
	}
	return impSpecs, nil
}

// verifySingleImportFromAllowedPackagesOnly returns true if the given import statement is allowed from the respective
// source package.
func verifySingleImportFromAllowedPackagesOnly(spec *ast.ImportSpec, packageName string, allowedPackages ...string) error {
	impPath, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return err
	}

	if err := checkForbidden(impPath, packageName); err != nil {
		return err
	}

	if !strings.HasPrefix(impPath, roxPrefix) {
		return nil
	}

	trimmed := strings.TrimPrefix(impPath, roxPrefix)

	for _, allowedPrefix := range allowedPackages {
		if strings.HasPrefix(trimmed, allowedPrefix) {
			return nil
		}
	}
	return fmt.Errorf("import %s is illegal", spec.Path.Value)
}

// checkForbidden returns an error if an import has been forbidden and the importing package isn't in the allowlist
func checkForbidden(impPath, packageName string) error {
	forbiddenDetails, ok := forbiddenImports[impPath]
	for !ok {
		if !stringutils.ConsumeSuffix(&impPath, "/...") {
			impPath += "/..."
		} else {
			slashIdx := strings.LastIndex(impPath, "/")
			if slashIdx == -1 {
				return nil
			}
			impPath = impPath[:slashIdx] + "/..."
		}
		forbiddenDetails, ok = forbiddenImports[impPath]
	}

	if forbiddenDetails.replacement == packageName {
		return nil
	}

	if forbiddenDetails.allowlist.Contains(packageName) {
		return nil
	}

	return fmt.Errorf("import is illegal; use %q instead", forbiddenDetails.replacement)
}

// verifyImportsFromAllowedPackagesOnly verifies that all Go files in (subdirectories of) root
// only import StackRox code from allowedPackages
func verifyImportsFromAllowedPackagesOnly(path, validImportRoot, packageName string) (errs []error) {
	imps, err := getImports(path)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "import retrieval"))
		return
	}

	allowedPackages := []string{validImportRoot, "generated", "image"}
	// The migrator is NOT allowed to import all code from pkg except process/id as that pkg is isolated.
	if validImportRoot != "pkg" && validImportRoot != "migrator" {
		allowedPackages = append(allowedPackages, "pkg")
	}
	// Specific sub-packages in pkg that the migrator is allowed to import go here.
	// Please be VERY prudent about adding to this list, since everything that's added to this list
	// will need to be protected by strict compatibility guarantees.
	if validImportRoot == "migrator" {
		allowedPackages = append(allowedPackages,
			"pkg/env", "pkg/rocksdb", "pkg/process/id", "pkg/migrations", "pkg/testutils", "pkg/batcher",
			"pkg/config", "pkg/features", "pkg/grpc/routes", "pkg/logging", "pkg/set", "pkg/version", "pkg/uuid",
			"pkg/utils", "pkg/fileutils", "pkg/buildinfo", "pkg/fsutils", "pkg/sliceutils")
	}

	if validImportRoot == "sensor/kubernetes" {
		allowedPackages = append(allowedPackages, "sensor/common")
	}

	// Allow scale tests to import some constants from central, to be more DRY.
	// This is not a problem since none of this code is used in prod anyway.
	if validImportRoot == "scale" {
		allowedPackages = append(allowedPackages, "central")
	}

	for _, imp := range imps {
		err := verifySingleImportFromAllowedPackagesOnly(imp, packageName, allowedPackages...)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "import verification for %s", imp.Path.Value))
		}
	}
	return
}

// Lifted straight from the goimports code
func isGoFile(f os.DirEntry) bool {
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

// Returns the list of go files in this directory (non recursively).
func getGoFilesInDir(packageDir string) (fileNames []string) {
	files, err := os.ReadDir(packageDir)
	if err != nil {
		logAndExit("Couldn't read go files in directory %s: %v", packageDir, err)
	}

	for _, file := range files {
		if !isGoFile(file) {
			continue
		}
		fileNames = append(fileNames, path.Join(packageDir, file.Name()))
	}
	return
}

func logAndExit(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func main() {
	var (
		goPath, goPathSet = os.LookupEnv("GOPATH")
		dirForPackageName func(packageName string) string
	)

	if goPathSet {
		p, err := filepath.Abs(goPath)
		if err != nil {
			logAndExit("failed to turn %s into an absolute path: %v", p, err)
		}
		goPath = p
	}

	cwd, err := os.Getwd()
	if err != nil {
		logAndExit("failed to query current working directory: %v", err)
	}

	if !goPathSet || !strings.HasPrefix(cwd, goPath) {
		dirForPackageName = func(packageName string) string {
			packageName = strings.TrimPrefix(strings.TrimPrefix(packageName, roxPrefix), stackroxPrefix)
			return path.Join(cwd, strings.TrimPrefix(packageName, roxPrefix))
		}
	} else {
		dirForPackageName = func(packageName string) string {
			packageName = strings.Replace(packageName, roxPrefix, stackroxPrefix, 1)
			return path.Join(goPath, "src", packageName)
		}
	}

	var failed bool

	for _, packageName := range os.Args[1:] {
		root, mustProcess := getRoot(packageName)
		if !mustProcess {
			continue
		}

		for _, goFile := range getGoFilesInDir(dirForPackageName(packageName)) {
			errs := verifyImportsFromAllowedPackagesOnly(goFile, root, packageName)
			if len(errs) > 0 {
				failed = true
				fmt.Printf("File %s\n", goFile)
				for _, err := range errs {
					fmt.Printf("\t%s\n", err)
				}
				fmt.Println()
			}
		}
	}

	if failed {
		logAndExit("Failures were found. Please fix your package imports!")
	}
}

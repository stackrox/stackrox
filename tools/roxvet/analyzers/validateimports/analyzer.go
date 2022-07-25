package validateimports

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = `check that imports are valid`

const roxPrefix = "github.com/stackrox/rox/"

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
		"sensor/kubernetes/sensor",
		"sensor/admission-control",
		"sensor/upgrader",
		"sensor/debugger",
		"sensor/tests",
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

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "validateimports",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Given the package name, get the root directory of the service.
// (The directory boundary that imports should not cross.)
func getRoot(packageName string) (root string, valid bool, err error) {
	if !strings.HasPrefix(packageName, roxPrefix) {
		return "", false, errors.Errorf("Package %s is not part of %s", packageName, roxPrefix)
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)

	for _, validRoot := range validRoots {
		if strings.HasPrefix(unqualifiedPackageName, validRoot) {
			return validRoot, true, nil
		}
	}

	// We explicitly ignore directories with Go files that we don't want to
	// lint, and exit with an error if the package doesn't match any of these directories.
	// This will make sure that this target throws an error if someone
	// adds a new service.
	for _, ignoredRoot := range ignoredRoots {
		if strings.HasPrefix(unqualifiedPackageName, ignoredRoot) {
			return "", false, nil
		}
	}

	return "", false, errors.Errorf("Package %s not found in list. If you added a new build root, "+
		"you might need to add it to the validRoots list in tools/roxvet/analyzers/validateimports/analyzer.go.", packageName)
}

// verifySingleImportFromAllowedPackagesOnly returns true if the given import statement is allowed from the respective
// source package.
func verifySingleImportFromAllowedPackagesOnly(spec *ast.ImportSpec, packageName string, importRoot string, allowedPackages ...string) error {
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
	return fmt.Errorf("%s cannot import from %s; only allowed roots are %+v", importRoot, trimmed, allowedPackages)
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
func verifyImportsFromAllowedPackagesOnly(pass *analysis.Pass, imports []*ast.ImportSpec, validImportRoot, packageName string) {
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
			"pkg/auth",
			"pkg/batcher",
			"pkg/bolthelper",
			"pkg/buildinfo",
			"pkg/concurrency",
			"pkg/config",
			"pkg/dackbox",
			"pkg/dackbox/crud",
			"pkg/dackbox/raw",
			"pkg/dackbox/sortedkeys",
			"pkg/db",
			"pkg/env",
			"pkg/features",
			"pkg/fileutils",
			"pkg/fsutils",
			"pkg/grpc/routes",
			"pkg/logging",
			"pkg/metrics",
			"pkg/migrations",
			"pkg/postgres/pgadmin",
			"pkg/postgres/pgconfig",
			"pkg/postgres/pgtest",
			"pkg/postgres/pgutils",
			"pkg/postgres/schema",
			"pkg/process/id",
			"pkg/retry",
			"pkg/rocksdb",
			"pkg/sac",
			"pkg/search",
			"pkg/search/postgres",
			"pkg/secondarykey",
			"pkg/set",
			"pkg/sliceutils",
			"pkg/sync",
			"pkg/testutils",
			"pkg/utils",
			"pkg/uuid",
			"pkg/version",
		)
	}

	if validImportRoot == "sensor/debugger" {
		allowedPackages = append(allowedPackages, "sensor/kubernetes/listener/resources")
	}

	if validImportRoot == "tools" {
		allowedPackages = append(allowedPackages, "central/globaldb", "central/metrics", "central/postgres", "central/role/resources",
			"sensor/kubernetes/sensor", "sensor/debugger")
	}

	if validImportRoot == "sensor/kubernetes" {
		allowedPackages = append(allowedPackages, "sensor/common")
	}

	// Allow scale tests to import some constants from central, to be more DRY.
	// This is not a problem since none of this code is used in prod anyway.
	if validImportRoot == "scale" {
		allowedPackages = append(allowedPackages, "central")
	}

	if validImportRoot == "sensor/tests" {
		allowedPackages = append(allowedPackages, "sensor/common", "sensor/kubernetes", "sensor/debugger")
	}

	for _, imp := range imports {
		err := verifySingleImportFromAllowedPackagesOnly(imp, packageName, validImportRoot, allowedPackages...)
		if err != nil {
			pass.Reportf(imp.Pos(), "invalid import %s: %v", imp.Path.Value, err)
		}
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	root, valid, err := getRoot(pass.Pkg.Path())
	if err != nil {
		pass.Reportf(token.NoPos, "couldn't find valid root: %v", err)
		return nil, nil
	}
	if !valid {
		return nil, nil
	}

	for _, file := range pass.Files {
		verifyImportsFromAllowedPackagesOnly(pass, file.Imports, root, pass.Pkg.Path())
	}

	return nil, nil
}

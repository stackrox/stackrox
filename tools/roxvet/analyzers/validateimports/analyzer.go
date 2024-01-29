package validateimports

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
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
	// Keep this in alphabetic order.
	validRoots = set.NewFrozenStringSet(
		"central",
		"compliance",
		"govulncheck",
		"image",
		"migrator",
		"migrator/migrations",
		"operator",
		"pkg",
		"roxctl",
		"scale",
		"scanner",
		"sensor/admission-control",
		"sensor/common",
		"sensor/debugger",
		"sensor/kubernetes",
		"sensor/tests",
		"sensor/testutils",
		"sensor/upgrader",
		"sensor/utils",
		"tools",
		"webhookserver",
		"qa-tests-backend/test-images/syslog",
	)

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
			replacement: "https://golang.org/doc/go1.18#ioutil",
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

type allowedPackage struct {
	path            string
	excludeChildren bool
}

func appendPackage(list []*allowedPackage, excludeChildren bool, pkgs ...string) []*allowedPackage {
	if list == nil {
		list = make([]*allowedPackage, len(pkgs))
	}

	for _, pkg := range pkgs {
		list = append(list, &allowedPackage{path: pkg, excludeChildren: excludeChildren})
	}
	return list
}

func appendPackageWithChildren(list []*allowedPackage, pkgs ...string) []*allowedPackage {
	return appendPackage(list, false, pkgs...)
}

func appendPackageWithoutChildren(list []*allowedPackage, pkgs ...string) []*allowedPackage {
	return appendPackage(list, true, pkgs...)
}

// Given the package name, get the root directory of the service.
// (The directory boundary that imports should not cross.)
func getRoot(packageName string) (root string, valid bool, err error) {
	if !strings.HasPrefix(packageName, roxPrefix) {
		return "", false, errors.Errorf("Package %s is not part of %s", packageName, roxPrefix)
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)
	pathElems := strings.Split(unqualifiedPackageName, string(filepath.Separator))
	for i := len(pathElems); i > 0; i-- {
		validRoot := strings.Join(pathElems[:i], string(filepath.Separator))
		if validRoots.Contains(validRoot) {
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
func verifySingleImportFromAllowedPackagesOnly(spec *ast.ImportSpec, packageName string, importRoot string, allowedPackages ...*allowedPackage) error {
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

	packagePaths := make([]string, 0, len(allowedPackages))
	for _, allowedPackage := range allowedPackages {
		if strings.HasPrefix(trimmed+"/", allowedPackage.path+"/") {
			if allowedPackage.excludeChildren && trimmed == allowedPackage.path {
				return nil
			}
			if !allowedPackage.excludeChildren {
				return nil
			}
		}
		packagePaths = append(packagePaths, allowedPackage.path)
	}
	return fmt.Errorf("%s cannot import from %s; only allowed roots are %+v", importRoot, trimmed, packagePaths)
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
	allowedPackages := []*allowedPackage{{path: validImportRoot}, {path: "generated"}, {path: "image"}}
	// The migrator is NOT allowed to import all codes from pkg except isolated packages.
	if validImportRoot != "pkg" && !strings.HasPrefix(validImportRoot, "migrator") {
		allowedPackages = appendPackageWithChildren(allowedPackages, "pkg")
	}

	// Specific sub-packages in pkg that the migrator and its sub-packages are allowed to import go here.
	// Please be VERY prudent about adding to this list, since everything that's added to this list
	// will need to be protected by strict compatibility guarantees.
	// Keep this in alphabetic order.
	if strings.HasPrefix(validImportRoot, "migrator") {
		allowedPackages = appendPackageWithChildren(allowedPackages,
			"pkg/auth",
			"pkg/batcher",
			"pkg/binenc",
			"pkg/bolthelper",
			"pkg/booleanpolicy/policyversion",
			"pkg/buildinfo",
			"pkg/concurrency",
			"pkg/config",
			"pkg/cve",
			"pkg/cvss/cvssv2",
			"pkg/cvss/cvssv3",
			"pkg/dackbox",
			"pkg/dackbox/crud",
			"pkg/dackbox/raw",
			"pkg/dackbox/sortedkeys",
			"pkg/db",
			"pkg/dberrors",
			"pkg/dbhelper",
			"pkg/defaults/policies",
			"pkg/env",
			"pkg/errorhelpers",
			"pkg/features",
			"pkg/fileutils",
			"pkg/fsutils",
			"pkg/grpc/routes",
			"pkg/images/types",
			"pkg/ioutils",
			"pkg/jsonutil",
			"pkg/logging",
			"pkg/mathutil",
			"pkg/metrics",
			"pkg/migrations",
			"pkg/nodes/converter",
			"pkg/policyutils",
			"pkg/postgres/gorm",
			"pkg/postgres/pgadmin",
			"pkg/postgres/pgconfig",
			"pkg/postgres/pgtest",
			"pkg/postgres/pgutils",
			"pkg/postgres/walker",
			"pkg/probeupload",
			"pkg/process/normalize",
			"pkg/process/id",
			"pkg/protoconv",
			"pkg/retry",
			"pkg/rocksdb",
			"pkg/sac",
			"pkg/scancomponent",
			"pkg/scans",
			"pkg/search",
			"pkg/search/postgres",
			"pkg/secondarykey",
			"pkg/set",
			"pkg/sliceutils",
			"pkg/stringutils",
			"pkg/sync",
			"pkg/testutils",
			"pkg/timestamp",
			"pkg/utils",
			"pkg/uuid",
			"pkg/version",
		)

		allowedPackages = appendPackageWithoutChildren(allowedPackages, "pkg/postgres")

		// Migrations shall not depend on current schemas. Each migration can include a copy of the schema before and
		// after a specific migration.
		if validImportRoot == "migrator" {
			allowedPackages = appendPackageWithChildren(allowedPackages, "pkg/postgres/schema")
		}

		if validImportRoot == "migrator/migrations" {
			allowedPackages = appendPackageWithChildren(allowedPackages, "migrator")
		}
	}

	if validImportRoot == "sensor/debugger" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/kubernetes/listener/resources", "sensor/kubernetes/client", "sensor/common/centralclient")
	}

	if validImportRoot == "tools" {
		allowedPackages = appendPackageWithChildren(allowedPackages,
			"central/globaldb", "central/metrics", "central/postgres", "pkg/sac/resources",
			"sensor/common/sensor", "sensor/common/centralclient", "sensor/kubernetes/client", "sensor/kubernetes/fake",
			"sensor/kubernetes/sensor", "sensor/debugger", "sensor/testutils",
			"compliance/collection/compliance", "compliance/collection/intervals")
	}

	if validImportRoot == "sensor/kubernetes" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/common")
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/utils")
	}

	// Allow scale tests to import some constants from central, to be more DRY.
	// This is not a problem since none of this code is used in prod anyway.
	if validImportRoot == "scale" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "central")
	}

	if validImportRoot == "sensor/tests" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/common", "sensor/kubernetes", "sensor/debugger", "sensor/testutils")
	}

	if validImportRoot == "sensor/common" {
		// Need this for unit tests.
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/debugger")
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

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/ast/astutil"
)

const roxPrefix = "github.com/stackrox/rox/"

var (
	validRoots = []string{
		"central",
		"migrator",
		"roxctl",
		"pkg",
		"sensor/kubernetes",
		"sensor/common",
		"tools",
		"integration-tests",
		"scale",
		"compliance",
		"webhookserver",
	}

	ignoredRoots = []string{
		"image",
		"generated",
		"tests",
	}

	forbiddenImports = map[string]string{
		"sync": "github.com/stackrox/rox/pkg/sync",
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
	fileContents, err := ioutil.ReadFile(path)
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, fileContents, parser.ImportsOnly)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't parse file %s", path)
	}
	impSections := astutil.Imports(fileSet, parsed)

	impSpecs := make([]*ast.ImportSpec, 0)
	for _, impSection := range impSections {
		for _, impSpec := range impSection {
			impSpecs = append(impSpecs, impSpec)
		}
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

	if replacement, ok := forbiddenImports[impPath]; ok && replacement != packageName {
		return fmt.Errorf("import is illegal; use %q instead", replacement)
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

// verifyImportsFromAllowedPackagesOnly verifies that all Go files in (subdirectories of) root
// only import StackRox code from allowedPackages
func verifyImportsFromAllowedPackagesOnly(path, validImportRoot, packageName string) (errs []error) {
	imps, err := getImports(path)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "import retrieval"))
		return
	}

	allowedPackages := []string{validImportRoot, "generated"}
	// The migrator is NOT allowed to import all code from pkg.
	if validImportRoot != "pkg" && validImportRoot != "migrator" {
		allowedPackages = append(allowedPackages, "pkg")
	}
	// Specific sub-packages in pkg that the migrator is allowed to import go here.
	// Please be VERY prudent about adding to this list, since everything that's added to this list
	// will need to be protected by strict compatibility guarantees.
	if validImportRoot == "migrator" {
		allowedPackages = append(allowedPackages, "pkg/migrations", "pkg/testutils")
	}

	// Allow central and cmd/deploy to import "image" (for fixtures)
	if validImportRoot == "central" || validImportRoot == "roxctl" {
		allowedPackages = append(allowedPackages, "image")
	}

	if validImportRoot == "sensor/kubernetes" {
		allowedPackages = append(allowedPackages, "sensor/common")
	}

	if validImportRoot == "cmd/roxdetect" {
		allowedPackages = append(allowedPackages, "cmd/common")
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
func isGoFile(f os.FileInfo) bool {
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

// Returns the list of go files in this package (non recursively).
func getGoFilesInPackage(goPath, packageName string) (fileNames []string) {
	dir := path.Join(goPath, "src", packageName)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logAndExit("Couldn't read go files in package %s: %s", packageName, err)
	}

	for _, file := range files {
		if !isGoFile(file) {
			continue
		}
		fileNames = append(fileNames, path.Join(dir, file.Name()))
	}
	return
}

func logAndExit(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

func main() {
	goPath, exists := os.LookupEnv("GOPATH")
	if !exists {
		logAndExit("GOPATH not found")
	}

	var failed bool

	for _, packageName := range os.Args[1:] {
		root, mustProcess := getRoot(packageName)
		if !mustProcess {
			continue
		}
		goFiles := getGoFilesInPackage(goPath, packageName)

		for _, goFile := range goFiles {
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

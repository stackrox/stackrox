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

	"golang.org/x/tools/go/ast/astutil"
)

const roxPrefix = "github.com/stackrox/rox/"

var (
	validRoots = []string{
		"benchmark-bootstrap",
		"benchmarks",
		"central",
		"cmd/base64",
		"cmd/deploy",
		"cmd/roxdetect",
		"pkg",
		"sensor/kubernetes",
		"sensor/swarm",
		"sensor/common",
		"tools",
		"integration-tests",
	}

	ignoredRoots = []string{
		"image",
		"tests",
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
		"you might need to add it to the validRoots list in tools/crosspkgimports/verify.go.", packageName)
	return "", false
}

// getImports parses the given Go file, returning its imports
func getImports(path string) ([]*ast.ImportSpec, error) {
	fileContents, err := ioutil.ReadFile(path)
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, fileContents, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse file %s: %s", path, err)
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

// roxImportsFromAllowedPackagesOnly returns true if imports to rox packages
// are made only from the allowed packages.
func roxImportsFromAllowedPackagesOnly(spec *ast.ImportSpec, allowedPackages ...string) (bool, error) {
	impPath, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return false, err
	}

	if !strings.HasPrefix(impPath, roxPrefix) {
		return true, nil
	}

	trimmed := strings.TrimPrefix(impPath, roxPrefix)

	for _, allowedPrefix := range allowedPackages {
		if strings.HasPrefix(trimmed, allowedPrefix) {
			return true, nil
		}
	}
	return false, nil
}

// verifyImportsFromAllowedPackagesOnly verifies that all Go files in (subdirectories of) root
// only import StackRox code from allowedPackages
func verifyImportsFromAllowedPackagesOnly(path, validImportRoot string) (errs []error) {
	imps, err := getImports(path)
	if err != nil {
		errs = append(errs, fmt.Errorf("import retrieval: %s", err))
		return
	}

	allowedPackages := []string{validImportRoot, "generated"}
	if validImportRoot != "pkg" {
		allowedPackages = append(allowedPackages, "pkg")
	}
	// Allow central to import "image" (for fixtures)
	if validImportRoot == "central" {
		allowedPackages = append(allowedPackages, "image")
	}

	if validImportRoot == "sensor/swarm" || validImportRoot == "sensor/kubernetes" {
		allowedPackages = append(allowedPackages, "sensor/common")
	}

	for _, imp := range imps {
		ok, err := roxImportsFromAllowedPackagesOnly(imp, allowedPackages...)
		if err != nil {
			errs = append(errs, fmt.Errorf("import verification for %s: %s", imp.Path.Value, err))
			continue
		}
		if !ok {
			errs = append(errs, fmt.Errorf("import %s is illegal", imp.Path.Value))
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
			errs := verifyImportsFromAllowedPackagesOnly(goFile, root)
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
		logAndExit("Failures were found. Please fix your cross-package imports!")
	}
}

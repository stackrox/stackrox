package framework

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/utils/strings/slices"
)

const (
	testFileGlobPattern = "*.test.yaml"
)

// Loader loads a test suite.
type Loader struct {
	rootDir            string
	additionalTestDirs []string
}

// NewLoader returns a loader and applies options to it.
func NewLoader(rootDir string, opts ...LoaderOpt) *Loader {
	loader := Loader{
		rootDir: rootDir,
	}

	for _, opt := range opts {
		opt(&loader)
	}
	return &loader
}

// LoaderOpt allows setting custom options.
type LoaderOpt func(loader *Loader)

// WithAdditionalTestDirs adds additional test source directories which are scanned for tests.
func WithAdditionalTestDirs(path ...string) LoaderOpt {
	return func(loader *Loader) {
		loader.additionalTestDirs = append(loader.additionalTestDirs, path...)
	}
}

// LoadSuite loads a helmtest suite from the given directory.
func (loader *Loader) LoadSuite() (*Test, error) {
	var suite Test
	if err := unmarshalYamlFromFileStrict(filepath.Join(loader.rootDir, "suite.yaml"), &suite); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "loading suite specification")
	}

	if suite.Name == "" {
		suite.Name = strings.TrimRight(loader.rootDir, "/")
	}

	testYAMLFiles, err := loader.readTestYAMLFiles()
	if err != nil {
		return nil, err
	}

	for _, file := range testYAMLFiles {
		test := Test{
			parent: &suite,
		}
		if err := unmarshalYamlFromFileStrict(file, &test); err != nil {
			return nil, errors.Wrapf(err, "loading test specification from file %s", file)
		}
		if test.Name == "" {
			test.Name = filepath.Base(file)
		}
		suite.Tests = append(suite.Tests, &test)
	}

	if err := suite.initialize(); err != nil {
		return nil, err
	}

	return &suite, nil
}

// readTestYAMLFiles locates test files, if any.
func (loader *Loader) readTestYAMLFiles() ([]string, error) {
	var testYAMLFiles []string
	var scannedDirs []string

	dirs := []string{loader.rootDir}
	dirs = append(dirs, loader.additionalTestDirs...)
	for _, dir := range dirs {
		if slices.Contains(scannedDirs, dir) {
			continue
		}

		dirGlob := filepath.Join(dir, testFileGlobPattern)

		yamlFiles, err := filepath.Glob(dirGlob)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving files using wildcard pattern %s", dirGlob)
		}
		testYAMLFiles = append(testYAMLFiles, yamlFiles...)
		scannedDirs = append(scannedDirs, dir)
	}

	return testYAMLFiles, nil
}

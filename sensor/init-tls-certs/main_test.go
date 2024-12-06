package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type initTLSCertsSuite struct {
	suite.Suite
}

func TestInitTLSCerts(t *testing.T) {
	suite.Run(t, new(initTLSCertsSuite))
}

func (s *initTLSCertsSuite) TestFileCopy() {
	tests := []struct {
		name           string
		setupDirs      func()
		expectedFiles  int
		expectedPrefix string
		expectedError  bool
	}{
		{
			name:           "Empty source dirs",
			setupDirs:      func() {},
			expectedFiles:  0,
			expectedPrefix: "",
			expectedError:  true,
		},
		{
			name: "Only legacySourceDir contains files",
			setupDirs: func() {
				s.createFiles(legacySourceDir, 3, "legacycert")
			},
			expectedFiles:  3,
			expectedPrefix: "legacycert",
			expectedError:  false,
		},
		{
			name: "Only newSourceDir contains files",
			setupDirs: func() {
				s.createFiles(newSourceDir, 3, "newcert")
			},
			expectedFiles:  3,
			expectedPrefix: "newcert",
			expectedError:  false,
		},
		{
			name: "Both legacySourceDir and newSourceDir contain files",
			setupDirs: func() {
				s.createFiles(legacySourceDir, 3, "legacycert")
				s.createFiles(newSourceDir, 3, "newcert")
			},
			expectedFiles:  3,
			expectedPrefix: "newcert", // prefer new certs if both exist
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.createDirs()
			tt.setupDirs()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			files, err := waitForSource(ctx)

			if tt.expectedError {
				s.Error(err)
				s.Nil(files)
			} else {
				s.Require().NoError(err)
				s.Len(files, tt.expectedFiles, "Expected %d certificate files", tt.expectedFiles)

				for _, file := range files {
					_, err := os.Stat(file)
					s.Require().NoError(err, "Expected file %s to exist", file)

					fileName := filepath.Base(file)
					s.True(strings.HasPrefix(fileName, tt.expectedPrefix), "Expected file name to start with %q, got %q", tt.expectedPrefix, fileName)
				}

				err = copyFiles(files, destinationDir)
				s.Require().NoError(err, "Failed to copy files")

				destFiles, err := os.ReadDir(destinationDir)
				s.Require().NoError(err, "Failed to read destination directory")
				s.Len(destFiles, tt.expectedFiles, "Expected %d files in destination directory", tt.expectedFiles)

				for _, file := range destFiles {
					fileName := file.Name()
					s.True(strings.HasPrefix(fileName, tt.expectedPrefix), "Expected file name to start with %q, got %q", tt.expectedPrefix, fileName)
				}
			}
		})
	}
}

func (s *initTLSCertsSuite) TestSanityCheckDestination() {
	tests := []struct {
		name           string
		destinationDir string
		expectedError  bool
	}{
		{
			name:           "Valid destination directory",
			destinationDir: s.T().TempDir(),
			expectedError:  false,
		},
		{
			name:           "Invalid destination directory",
			destinationDir: "/non/existent/dir",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			destinationDir = tt.destinationDir
			result, err := sanityCheckDestination()

			if tt.expectedError {
				s.Error(err)
				s.Empty(result, "Expected empty result on error")
			} else {
				s.NoError(err)
				s.NotEmpty(result, "Expected non-empty result")
			}
		})
	}
}

func (s *initTLSCertsSuite) createFiles(dir string, count int, prefix string) {
	for i := 0; i < count; i++ {
		_, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s%d", prefix, i)))
		s.Require().NoError(err, "Failed to create cert file in %s", dir)
	}
}

func (s *initTLSCertsSuite) createDirs() {
	legacySourceDir = s.T().TempDir()
	newSourceDir = s.T().TempDir()
	destinationDir = s.T().TempDir()
}

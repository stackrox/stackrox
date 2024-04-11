package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

func cleanup(t *testing.T, tmpPath string) {
	err := os.RemoveAll(tmpPath)
	assert.NoError(t, err)
}

func TestDefaultIO(t *testing.T) {
	t.Run("default environment", func(t *testing.T) {
		defaultIO := DefaultIO()
		assert.NotNil(t, defaultIO)

		outImpl := defaultIO.Out().(*os.File)
		assert.NotNil(t, outImpl)
		assert.Equal(t, "/dev/stdout", outImpl.Name())

		errImpl := defaultIO.ErrOut().(*os.File)
		assert.NotNil(t, errImpl)
		assert.Equal(t, "/dev/stderr", errImpl.Name())
	})

	tmpPath, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer cleanup(t, tmpPath)

	writeableSubDirPath := filepath.Join(tmpPath, "writeable-dir")
	err = os.Mkdir(writeableSubDirPath, 0777)
	assert.NoError(t, err)

	writeableFilePath := filepath.Join(tmpPath, "writeable-file")
	err = os.WriteFile(writeableFilePath, []byte{}, 0600)
	assert.NoError(t, err)

	missingFileInWriteableSubDirPath := filepath.Join(writeableSubDirPath, "new-file")

	writeableFileWithMissingSubDirPath := filepath.Join(writeableSubDirPath, "missing-dir", "writeable-file")

	notWriteableFilePath := filepath.Join(tmpPath, "not-writeable-file")
	err = os.WriteFile(notWriteableFilePath, []byte{}, 0400)
	assert.NoError(t, err)

	const stdoutPath = "/dev/stdout"
	const stderrPath = "/dev/stderr"

	testCases := []struct {
		testName            string
		outFilePath         string
		expectedOutFileName string
		errFilePath         string
		expectedErrFileName string
	}{
		{
			testName:            "out and err point to an existing writeable file",
			outFilePath:         writeableFilePath,
			expectedOutFileName: writeableFilePath,
			errFilePath:         writeableFilePath,
			expectedErrFileName: writeableFilePath,
		},
		{
			testName:            "out and err variables are empty",
			outFilePath:         "",
			expectedOutFileName: stdoutPath,
			errFilePath:         "",
			expectedErrFileName: stderrPath,
		},
		{
			testName:            "out and err point to missing files with writeable path base -> files are created",
			outFilePath:         missingFileInWriteableSubDirPath,
			expectedOutFileName: missingFileInWriteableSubDirPath,
			errFilePath:         writeableFileWithMissingSubDirPath,
			expectedErrFileName: writeableFileWithMissingSubDirPath,
		},
		{
			testName:            "out and err point to a non-writeable file -> use defaults",
			outFilePath:         notWriteableFilePath,
			expectedOutFileName: stdoutPath,
			errFilePath:         notWriteableFilePath,
			expectedErrFileName: stderrPath,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			t.Setenv(env.OutputFile.EnvVar(), testCase.outFilePath)
			t.Setenv(env.ErrorFile.EnvVar(), testCase.errFilePath)
			defaultIO := DefaultIO()

			outImpl := defaultIO.Out().(*os.File)
			assert.NotNil(t, outImpl)
			assert.Equal(t, testCase.expectedOutFileName, outImpl.Name())
			if testCase.outFilePath == testCase.expectedOutFileName {
				_, err = os.Stat(testCase.outFilePath)
				assert.NoError(t, err)
			}

			errImpl := defaultIO.ErrOut().(*os.File)
			assert.NotNil(t, errImpl)
			assert.Equal(t, testCase.expectedErrFileName, errImpl.Name())
			if testCase.errFilePath == testCase.expectedErrFileName {
				_, err = os.Stat(testCase.errFilePath)
				assert.NoError(t, err)
			}
		})
	}

	err = os.Chmod(notWriteableFilePath, 0666)
	assert.NoError(t, err)
}

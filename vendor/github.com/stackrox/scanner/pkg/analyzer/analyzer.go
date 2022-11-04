package analyzer

import (
	"io"
	"os"

	"github.com/stackrox/scanner/pkg/component"
	"github.com/stackrox/scanner/pkg/elf"
)

const (
	// DefaultMaxELFExecutableFileSizeMB is the default value for the max ELF executable file we analyze.
	DefaultMaxELFExecutableFileSizeMB = 800

	// DefaultMaxLazyReaderBufferSizeMB is the default maximum lazy reader buffer size. Any file data beyond this
	// limit is backed by temporary files on disk.
	DefaultMaxLazyReaderBufferSizeMB = 100

	// DefaultMaxExtractableFileSizeMB is the default value for the max extractable file size.
	DefaultMaxExtractableFileSizeMB = 200

	// ElfHeaderSize is the size of an ELF header based on https://refspecs.linuxfoundation.org/elf/gabi4+/ch4.eheader.html.
	ElfHeaderSize = 16
)

var (
	maxExtractableFileSize   int64 = DefaultMaxExtractableFileSizeMB * 1024 * 1024
	maxELFExecutableFileSize int64 = DefaultMaxELFExecutableFileSizeMB * 1024 * 1024
	maxLazyReaderBufferSize  int64 = DefaultMaxLazyReaderBufferSizeMB * 1024 * 1024
)

// SetMaxExtractableFileSize sets the max extractable file size. It is NOT
// thread-safe, and callers must ensure that it is called only when no scans are
// in progress (ex: during initialization). See comments on the
// maxExtractableFileSize variable for more details on its purpose.
func SetMaxExtractableFileSize(val int64) {
	maxExtractableFileSize = val
}

// GetMaxExtractableFileSize returns the maximum size of a single file within a
// tarball that will be extracted. This protects against malicious files that may
// be used in an attempt to perform a Denial of Service attack.
func GetMaxExtractableFileSize() int64 {
	return maxExtractableFileSize
}

// GetMaxLazyReaderBufferSize returns the maximum lazy reader buffer size. Any
// file data beyond this limit is backed by temporary files on disk.
func GetMaxLazyReaderBufferSize() int64 {
	return maxLazyReaderBufferSize
}

// SetMaxLazyReaderBufferSize sets the max lazy reader buffer size. It is NOT
// thread-safe, and callers must ensure that it is called only when no scans are
// in progress (ex: during initialization). See comments on the
// maxLazyReaderBufferSize variable for more details on its purpose.
func SetMaxLazyReaderBufferSize(val int64) {
	maxLazyReaderBufferSize = val
}

// GetMaxELFExecutableFileSize returns the maximum size of an ELF executable file
// tarball that will be analyzed.
func GetMaxELFExecutableFileSize() int64 {
	return maxELFExecutableFileSize
}

// SetMaxELFExecutableFileSize sets the max ELF executable file size. It is NOT
// thread-safe, and callers must ensure that it is called only when no scans are
// in progress (ex: during initialization). See comments on the
// maxELFExecutableFileSize variable for more details on its purpose.
func SetMaxELFExecutableFileSize(val int64) {
	maxELFExecutableFileSize = val
}

// Analyzer defines the functions for analyzing images and extracting the components present in them.
type Analyzer interface {
	ProcessFile(filePath string, fi os.FileInfo, contents io.ReaderAt) []*component.Component
}

// Files stores information on a sub-set of files being analyzed.
// It provides methods to retrieve information from individual files, or list
// them based on some prefix.
type Files interface {
	// Get returns the data about a file if it exists, otherwise set exists to false.
	Get(path string) (data FileData, exists bool)

	// GetFilesPrefix returns a map of files matching the specified prefix, empty map
	// if none found. The prefix itself is not matched.
	GetFilesPrefix(prefix string) (filesMap map[string]FileData)
}

// FileData is the contents of a file and relevant metadata.
type FileData struct {
	// Contents is the contents of the file.
	Contents []byte

	// Executable indicates if the file is executable.
	Executable bool

	// ELFMetadata contains the dynamic library dependency metadata if the file is in ELF format.
	ELFMetadata *elf.Metadata
}

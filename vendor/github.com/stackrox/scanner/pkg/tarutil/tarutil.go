// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tarutil implements some tar utility functions.
package tarutil

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os/exec"
	"path"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/scanner/pkg/analyzer"
	"github.com/stackrox/scanner/pkg/elf"
	"github.com/stackrox/scanner/pkg/ioutils"
	"github.com/stackrox/scanner/pkg/matcher"
	"github.com/stackrox/scanner/pkg/metrics"
)

var (
	readLen           = 6 // max bytes to sniff
	gzipHeader        = []byte{0x1f, 0x8b}
	bzip2Header       = []byte{0x42, 0x5a, 0x68}
	xzHeader          = []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}
	shebangHeader     = []byte{0x23, 0x21}
	shebangHeaderSize = len(shebangHeader)
)

// ExtractFiles decompresses and extracts only the specified files from an
// io.Reader representing an archive.
func ExtractFiles(r io.Reader, filenameMatcher matcher.Matcher) (LayerFiles, error) {
	files := CreateNewLayerFiles(nil)

	// Decompress the archive.
	tr, err := NewTarReadCloser(r)
	if err != nil {
		return files, errors.Wrap(err, "could not extract tar archive")
	}
	defer tr.Close()

	// Telemetry variables.
	var numFiles, numMatchedFiles, numExtractedContentBytes int

	var prevLazyReader ioutils.LazyReaderAtWithDiskBackedBuffer
	defer func() {
		if prevLazyReader != nil {
			utils.IgnoreError(prevLazyReader.Close)
		}
	}()

	// For each element in the archive
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return files, errors.Wrap(err, "could not advance in the tar archive")
		}
		numFiles++

		// Get element filename
		filename := strings.TrimPrefix(hdr.Name, "./")

		var contents io.ReaderAt
		if hdr.FileInfo().Mode().IsRegular() {
			// Recycle the buffer, if possible.
			var buf []byte
			if prevLazyReader != nil {
				buf = prevLazyReader.StealBuffer()
				utils.IgnoreError(prevLazyReader.Close)
			}
			prevLazyReader = ioutils.NewLazyReaderAtWithDiskBackedBuffer(tr, hdr.Size, buf, analyzer.GetMaxLazyReaderBufferSize())
			contents = prevLazyReader
		} else {
			contents = bytes.NewReader(nil)
		}

		match, extractContents := filenameMatcher.Match(filename, hdr.FileInfo(), contents)
		if !match {
			continue
		}
		numMatchedFiles++

		// File size limit
		if extractContents && hdr.Size > analyzer.GetMaxExtractableFileSize() {
			log.Errorf("Skipping file %q (%d bytes) because it was greater than the configured maxExtractableFileSize of %d MiB",
				filename, hdr.Size, analyzer.GetMaxExtractableFileSize()/1024/1024)
			continue
		}

		// Extract the element
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeLink:
			var fileData analyzer.FileData

			executable := matcher.IsFileExecutable(hdr.FileInfo())
			if hdr.Size > analyzer.GetMaxELFExecutableFileSize() {
				log.Warnf("Skipping ELF executable check for file %q (%d bytes) because it is larger than the configured maxELFExecutableFileSize of %d MiB",
					filename, hdr.Size, analyzer.GetMaxELFExecutableFileSize()/1024/1024)
			} else {
				if hdr.Size >= analyzer.ElfHeaderSize { // Only bother attempting to get ELF metadata if the file is large enough for the ELF header.
					fileData.ELFMetadata, err = elf.GetExecutableMetadata(contents)
					if err != nil {
						log.Errorf("Failed to get dependencies for %s: %v", filename, err)
					}
				}
				if executable && hdr.Typeflag != tar.TypeLink && fileData.ELFMetadata == nil {
					// If the type is a hard link then we will not be able to read it.
					// Keep it as an executable in order to not introduce false negatives.
					shebangBytes := make([]byte, shebangHeaderSize)
					if hdr.Size > int64(shebangHeaderSize) {
						_, err := contents.ReadAt(shebangBytes, 0)
						if err != nil {
							log.Errorf("unable to read first two bytes of file %s: %v", filename, err)
							continue
						}
					}
					executable = bytes.Equal(shebangBytes, shebangHeader)
				}
			}

			if extractContents {
				if hdr.Typeflag == tar.TypeLink {
					// A hard-link necessarily points to a previous absolute path in the
					// archive which we look if it was already extracted.
					linkedFile, ok := files.data[hdr.Linkname]
					if ok {
						fileData.Contents = linkedFile.Contents
					}
				} else {
					d := make([]byte, hdr.Size)
					if nRead, err := contents.ReadAt(d, 0); err != nil {
						log.Errorf("error reading %q: %v", hdr.Name, err)
						d = d[:nRead]
					}

					// Put the file directly
					fileData.Contents = d
					numExtractedContentBytes += len(d)
				}
			}
			fileData.Executable = executable
			files.data[filename] = fileData
		case tar.TypeSymlink:
			if path.IsAbs(hdr.Linkname) {
				files.links[filename] = path.Clean(hdr.Linkname)[1:]
			} else {
				files.links[filename] = path.Clean(path.Join(path.Dir(filename), hdr.Linkname))
			}
		case tar.TypeDir:
			// Do not bother saving the contents,
			// and directories are NOT considered executable.
			// However, add to the map, so the entry will exist.
			files.data[filename] = analyzer.FileData{}
		}
	}
	files.detectRemovedFiles()

	metrics.ObserveFileCount(numFiles)
	metrics.ObserveMatchedFileCount(numMatchedFiles)
	metrics.ObserveExtractedContentBytes(numExtractedContentBytes)

	return files, nil
}

// XzReader implements io.ReadCloser for data compressed via `xz`.
type XzReader struct {
	io.ReadCloser
	cmd     *exec.Cmd
	closech chan error
}

// NewXzReader returns an io.ReadCloser by executing a command line `xz`
// executable to decompress the provided io.Reader.
//
// It is the caller's responsibility to call Close on the XzReader when done.
func NewXzReader(r io.Reader) (*XzReader, error) {
	rpipe, wpipe := io.Pipe()
	ex, err := exec.LookPath("xz")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(ex, "--decompress", "--stdout")

	closech := make(chan error)

	cmd.Stdin = r
	cmd.Stdout = wpipe

	go func() {
		err := cmd.Run()
		wpipe.CloseWithError(err)
		closech <- err
	}()

	return &XzReader{rpipe, cmd, closech}, nil
}

// Close cleans up the resources used by an XzReader.
func (r *XzReader) Close() error {
	r.ReadCloser.Close()
	r.cmd.Process.Kill()
	return <-r.closech
}

// TarReadCloser embeds a *tar.Reader and the related io.Closer
// It is the caller's responsibility to call Close on TarReadCloser when
// done.
type TarReadCloser struct {
	*tar.Reader
	io.Closer
}

// Close cleans up the resources used by a TarReadCloser.
func (r *TarReadCloser) Close() error {
	return r.Closer.Close()
}

// NewTarReadCloser attempts to detect the compression algorithm for an
// io.Reader and returns a TarReadCloser wrapping the Reader to transparently
// decompress the contents.
//
// Gzip/Bzip2/XZ detection is done by using the magic numbers:
// Gzip: the first two bytes should be 0x1f and 0x8b. Defined in the RFC1952.
// Bzip2: the first three bytes should be 0x42, 0x5a and 0x68. No RFC.
// XZ: the first three bytes should be 0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00. No RFC.
func NewTarReadCloser(r io.Reader) (*TarReadCloser, error) {
	br := bufio.NewReader(r)
	header, err := br.Peek(readLen)
	if err == nil {
		switch {
		case bytes.HasPrefix(header, gzipHeader):
			gr, err := gzip.NewReader(br)
			if err != nil {
				return nil, err
			}
			return &TarReadCloser{tar.NewReader(gr), gr}, nil
		case bytes.HasPrefix(header, bzip2Header):
			bzip2r := io.NopCloser(bzip2.NewReader(br))
			return &TarReadCloser{tar.NewReader(bzip2r), bzip2r}, nil
		case bytes.HasPrefix(header, xzHeader):
			xzr, err := NewXzReader(br)
			if err != nil {
				return nil, err
			}
			return &TarReadCloser{tar.NewReader(xzr), xzr}, nil
		}
	}

	dr := io.NopCloser(br)
	return &TarReadCloser{tar.NewReader(dr), dr}, nil
}

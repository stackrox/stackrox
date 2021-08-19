package util

import (
	"bytes"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/helm/util/internal/ignore"
	"github.com/stackrox/rox/pkg/stringutils"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type fakeFileInfo string

func (i fakeFileInfo) Name() string {
	return path.Base(string(i))
}

func (i fakeFileInfo) Size() int64 {
	return 0
}

func (i fakeFileInfo) Mode() os.FileMode {
	if i.IsDir() {
		return os.ModeDir | 0700
	}
	return 0600
}

func (i fakeFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (i fakeFileInfo) IsDir() bool {
	return strings.HasSuffix(string(i), "/")
}

func (i fakeFileInfo) Sys() interface{} {
	return nil
}

// FilterOptions allows customizing the filtering behavior.
type FilterOptions struct {
	// The name of the ignore file. If empty, it will be assumed that no ignorefile exists.
	// The syntax of this file is the same as for standard .helmignore.
	IgnoreFileName string
	// If set to true, apply default Helm filtering rules in the absence of an ignorefile.
	// Otherwise, if no ignorefile exists, return the original set of files.
	ApplyDefaultIgnoreRules bool
	// If set to true, the ignorefile will be retained in the output, otherwise it will be
	// suprressed.
	KeepIgnoreFile bool
}

var (
	helmFilterOpts = FilterOptions{
		IgnoreFileName:          ignore.HelmIgnore,
		ApplyDefaultIgnoreRules: true,
		KeepIgnoreFile:          true,
	}
)

// FilterFilesWithOptions filters the given list of files, according to the specified options.
func FilterFilesWithOptions(files []*loader.BufferedFile, opts FilterOptions) ([]*loader.BufferedFile, error) {
	sortedFiles := make([]*loader.BufferedFile, len(files))
	copy(sortedFiles, files)

	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Name < sortedFiles[j].Name
	})

	var ignoreFile *loader.BufferedFile
	if opts.IgnoreFileName != "" {
		for _, file := range sortedFiles {
			if file.Name == opts.IgnoreFileName {
				ignoreFile = file
				break
			}
			if file.Name > opts.IgnoreFileName {
				// Lexicographical sorting ensures we can stop here.
				break
			}
		}
	}

	var ignoreRules *ignore.Rules
	if ignoreFile == nil {
		if !opts.ApplyDefaultIgnoreRules {
			return sortedFiles, nil // nothing is ignored
		}
		ignoreRules = ignore.Empty()
		ignoreRules.AddDefaults()
	} else {
		var err error
		ignoreRules, err = ignore.Parse(bytes.NewReader(ignoreFile.Data))
		if err != nil {
			return nil, errors.Wrapf(err, "parsing rules from ignorefile %s", opts.IgnoreFileName)
		}
	}

	// This is always a copy, so no harm in overwriting.
	filtered := sortedFiles[:0]

	var skipPrefix, prevDir string
	for _, file := range sortedFiles {
		if file == ignoreFile {
			if opts.KeepIgnoreFile {
				filtered = append(filtered, file)
			}
			continue
		}
		if skipPrefix != "" && strings.HasPrefix(file.Name, skipPrefix) {
			// skipPrefix is guaranteed to always end with a slash (see below), so we'll only
			// skip at a per-directory level.
			continue
		}

		skipPrefix = ""

		// Ensure that currParentDir ends in a slash (or is empty). To do so, trim it after the last
		// slash, but only if that slash is not the last character in currParentDir.
		currParentDir := stringutils.LongestCommonPrefixUTF8(prevDir, file.Name)
		if lastSlash := strings.LastIndex(currParentDir, "/"); lastSlash != -1 && lastSlash != len(currParentDir)-1 {
			currParentDir = currParentDir[:lastSlash+1]
		}

		for skipPrefix == "" {
			nextSlash := strings.IndexRune(file.Name[len(currParentDir):], '/')
			if nextSlash == -1 {
				break
			}
			currParentDir = file.Name[:len(currParentDir)+nextSlash+1]
			if ignoreRules.Ignore(currParentDir[:len(currParentDir)-1], fakeFileInfo(currParentDir)) {
				skipPrefix = currParentDir
			}
		}
		if skipPrefix != "" {
			// Ignore an entire directory (including the one the current file is in).
			continue
		}

		prevDir = currParentDir
		if ignoreRules.Ignore(file.Name, fakeFileInfo(file.Name)) {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered, nil
}

// FilterFiles filters the given files, treating them as a Helm chart (i.e., using
// `.helmignore` as the name of the ignore file, and applying the default exclusion rule of
// excluding dotfiles in templates/ in the absence of an ignorefile).
func FilterFiles(files []*loader.BufferedFile) ([]*loader.BufferedFile, error) {
	return FilterFilesWithOptions(files, helmFilterOpts)
}

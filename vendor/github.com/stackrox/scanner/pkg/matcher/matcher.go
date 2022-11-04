package matcher

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/pkg/whiteout"
)

// Matcher defines the functions necessary for matching files.
type Matcher interface {
	// Match determines if the given file, identified by the given path and info,
	// matches.
	// The first return indicates if it matches.
	// The second return indicates if the contents of the file should be saved (if the first value is true).
	Match(fullPath string, fileInfo os.FileInfo, contents io.ReaderAt) (matches bool, extract bool)
}

// PrefixMatcher is a matcher that uses file prefixes.
type PrefixMatcher interface {
	Matcher

	// GetCommonPrefixDirs list all directories from all the prefixes used in this
	// matcher, and returns a list of common directories in all of them, up to one
	// level below the root dir, e.g. prefixes are {"a/b/f", "a/c/f", "b/c/"} the
	// common prefix list is {"a/", "b/c/"}. The returned directories will always be
	// terminated with /. If a name is not terminated by a slash it is considered a
	// file and ignored. Example:
	//
	// Prefixes:
	//   - var/lib/rpm/
	//   - var/lib/dpkg/
	//   - root/buildinfo/
	//   - usr/bin
	//   - usr/bin/bash
	//   - etc/apt.sources
	//
	// Output:
	//   - var/lib/
	//   - root/buildinfo/
	//   - usr/
	//   - etc/
	GetCommonPrefixDirs() []string
}

type allowlistMatcher struct {
	allowlist []string
}

// NewPrefixAllowlistMatcher returns a prefix matcher that matches all filenames
// which have any of the passed paths as a prefix.
func NewPrefixAllowlistMatcher(allowlist ...string) PrefixMatcher {
	return &allowlistMatcher{allowlist: allowlist}
}

func (m *allowlistMatcher) Match(fullPath string, _ os.FileInfo, _ io.ReaderAt) (matches bool, extract bool) {
	for _, s := range m.allowlist {
		if strings.HasPrefix(fullPath, s) {
			return true, true
		}
	}
	return false, false
}

func (m *allowlistMatcher) GetCommonPrefixDirs() []string {
	return findCommonDirPrefixes(m.allowlist)
}

// findCommonDirPrefixes goes over all prefixes, steps one level down from the
// root directory, and returns exactly one common prefix per first level dir
// referenced. It does it by doing creating a trie-like structure with the
// directory tree filtering paths with only single-children nodes.
func findCommonDirPrefixes(prefixes []string) []string {
	prefixToSubdirs := make(map[string]set.StringSet)
	for _, d := range prefixes {
		for d != "" {
			p, _ := path.Split(strings.TrimSuffix(d, "/"))
			s := prefixToSubdirs[p]
			s.Add(d)
			prefixToSubdirs[p] = s
			d = p
		}
	}
	// Work on one step below root.
	firstLevelDirs := prefixToSubdirs[""].AsSlice()
	ret := firstLevelDirs[:0]
	for _, d := range firstLevelDirs {
		for len(prefixToSubdirs[d]) == 1 {
			d = prefixToSubdirs[d].GetArbitraryElem()
		}
		d, _ := path.Split(d)
		ret = append(ret, d)
	}
	return ret
}

type whiteoutMatcher struct{}

// NewWhiteoutMatcher returns a matcher that matches all whiteout files
// (ie files which have been deleted) and opaque directories.
func NewWhiteoutMatcher() Matcher {
	return &whiteoutMatcher{}
}

func (w *whiteoutMatcher) Match(fullPath string, _ os.FileInfo, _ io.ReaderAt) (matches bool, extract bool) {
	basePath := filepath.Base(fullPath)
	return strings.HasPrefix(basePath, whiteout.Prefix), false
}

type executableMatcher struct{}

// NewExecutableMatcher returns a matcher that matches all executable regular files.
func NewExecutableMatcher() Matcher {
	return &executableMatcher{}
}

func (e *executableMatcher) Match(_ string, fi os.FileInfo, _ io.ReaderAt) (matches bool, extract bool) {
	return IsFileExecutable(fi), false
}

type regexpMatcher struct {
	expr        *regexp.Regexp
	extractable bool
}

// NewRegexpMatcher returns a matcher that matches all files which adhere to the given regexp pattern.
func NewRegexpMatcher(expr *regexp.Regexp, extractable bool) Matcher {
	return &regexpMatcher{
		expr:        expr,
		extractable: extractable,
	}
}

func (r *regexpMatcher) Match(fullPath string, _ os.FileInfo, _ io.ReaderAt) (matches bool, extract bool) {
	if r.expr.MatchString(fullPath) {
		return true, r.extractable
	}

	return false, false
}

type symlinkMatcher struct{}

// NewSymbolicLinkMatcher returns a matcher that matches symbolic links
func NewSymbolicLinkMatcher() Matcher {
	return &symlinkMatcher{}
}

func (o *symlinkMatcher) Match(_ string, fileInfo os.FileInfo, _ io.ReaderAt) (matches bool, extract bool) {
	return fileInfo.Mode()&fs.ModeSymlink != 0, false
}

type orMatcher struct {
	matchers []Matcher
}

// NewOrMatcher returns a matcher that matches if any of the passed sub-matchers does.
func NewOrMatcher(subMatchers ...Matcher) Matcher {
	return &orMatcher{matchers: subMatchers}
}

func (o *orMatcher) Match(fullPath string, fileInfo os.FileInfo, contents io.ReaderAt) (matches bool, extract bool) {
	for _, subMatcher := range o.matchers {
		if matches, extractable := subMatcher.Match(fullPath, fileInfo, contents); matches {
			return true, extractable
		}
	}
	return false, false
}

type andMatcher struct {
	matchers []Matcher
}

// NewAndMatcher returns a matcher that matches if all the passed sub-matchers match.
func NewAndMatcher(subMatchers ...Matcher) Matcher {
	return &andMatcher{matchers: subMatchers}
}

func (a *andMatcher) Match(fullPath string, fileInfo os.FileInfo, contents io.ReaderAt) (matches bool, extract bool) {
	if len(a.matchers) == 0 {
		return false, false
	}

	extract = true
	for _, subMatcher := range a.matchers {
		match, extractable := subMatcher.Match(fullPath, fileInfo, contents)
		if !match {
			return false, false
		}
		extract = extract && extractable
	}
	return true, extract
}

// IsFileExecutable returns if the file is an executable regular file.
func IsFileExecutable(fileInfo fs.FileInfo) bool {
	return fileInfo.Mode().IsRegular() && fileInfo.Mode()&0111 != 0
}

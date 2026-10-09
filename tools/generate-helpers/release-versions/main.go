// release-versions generates initial-release database sequences from Git tags.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/version/productstreams"
)

const sequencePath = "pkg/migrations/internal/seq_num.go"

var initialRelease = regexp.MustCompile(`^[1-9][0-9]*\.(0|[1-9][0-9]*)\.0(-rc\.(0|[1-9][0-9]*))?$`)
var targetVersion = regexp.MustCompile(`^[1-9][0-9]*\.(0|[1-9][0-9]*)\.(x|0|[1-9][0-9]*)(-[A-Za-z0-9.-]+)?$`)

func main() {
	target := flag.String("target", "", "target version (defaults to make tag)")
	flag.Parse()
	if err := run(*target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func command(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %v: %w: %s", name, args, err, stderr.String())
	}
	return string(data), nil
}

func run(target string) error {
	root, err := command("", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("find repository root: %w", err)
	}
	dir := strings.TrimSpace(root)
	if target == "" {
		data, err := command(dir, "make", "--quiet", "--no-print-directory", "tag")
		if err != nil {
			return fmt.Errorf("determine target version with make tag: %w", err)
		}
		target = strings.TrimSpace(data)
	}
	data, err := generate(dir, target)
	if err != nil {
		return fmt.Errorf("generate release versions: %w", err)
	}
	outputPath := filepath.Join(dir, "pkg/migrations/release_versions.go")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write generated release versions to %s: %w", outputPath, err)
	}
	return nil
}

func generate(dir, target string) ([]byte, error) {
	targetXy, err := parseTargetStream(target)
	if err != nil {
		return nil, err
	}
	tags, err := listReleaseTags(dir)
	if err != nil {
		return nil, err
	}
	releaseTags, err := selectReleaseTags(tags, targetXy)
	if err != nil {
		return nil, err
	}
	sequences, err := readReleaseSequences(dir, releaseTags)
	if err != nil {
		return nil, err
	}
	if err := ensureTargetSequence(dir, target, targetXy, sequences); err != nil {
		return nil, err
	}
	streams, err := validateReleaseSequences(target, targetXy, sequences)
	if err != nil {
		return nil, err
	}
	return renderReleaseVersions(streams, sequences)
}

func parseTargetStream(target string) (productstreams.XYVersion, error) {
	if !targetVersion.MatchString(target) {
		return productstreams.XYVersion{}, fmt.Errorf("invalid target version %q", target)
	}
	targetXy, err := productstreams.ParseXYFromVersionString(target)
	if err != nil {
		return productstreams.XYVersion{}, fmt.Errorf("parse target stream from %q: %w", target, err)
	}
	if err := validateReleaseStream(targetXy); err != nil {
		return productstreams.XYVersion{}, fmt.Errorf("validate target stream %s: %w", targetXy, err)
	}
	return targetXy, nil
}

func listReleaseTags(dir string) (string, error) {
	shallow, err := command(dir, "git", "rev-parse", "--is-shallow-repository")
	if err != nil {
		return "", fmt.Errorf("check whether checkout is shallow: %w", err)
	}
	if strings.TrimSpace(shallow) != "false" {
		return "", errors.New("release generation requires a full checkout with release tags")
	}
	tags, err := command(dir, "git", "tag", "--list",
		"[0-9]*.[0-9]*.0",
		"[0-9]*.[0-9]*.0-rc.[0-9]*",
	)
	if err != nil {
		return "", fmt.Errorf("list release tags: %w", err)
	}
	return tags, nil
}

func selectReleaseTags(tags string, targetXy productstreams.XYVersion) (map[productstreams.XYVersion]string, error) {
	releaseTags := make(map[productstreams.XYVersion]string)
	for tag := range strings.FieldsSeq(tags) {
		if !initialRelease.MatchString(tag) {
			continue
		}
		stream, err := productstreams.ParseXYFromVersionString(tag)
		if err != nil {
			return nil, fmt.Errorf("parse release stream from tag %q: %w", tag, err)
		}
		if stream.Compare(productstreams.XYVersion{X: 4, Y: 4}) < 0 || stream.Compare(targetXy) > 0 {
			continue
		}
		if err := validateReleaseStream(stream); err != nil {
			return nil, fmt.Errorf("validate historical release stream %s: %w", stream, err)
		}
		if selected, exists := releaseTags[stream]; !exists || newerInitialRelease(tag, selected) {
			releaseTags[stream] = tag
		}
	}
	return releaseTags, nil
}

func readReleaseSequences(dir string, releaseTags map[productstreams.XYVersion]string) (map[productstreams.XYVersion]int, error) {
	sequences := make(map[productstreams.XYVersion]int)
	for stream, tag := range releaseTags {
		data, err := command(dir, "git", "show", "refs/tags/"+tag+":"+sequencePath)
		if err != nil {
			return nil, fmt.Errorf("read %s from release tag %s: %w", sequencePath, tag, err)
		}
		sequence, err := parseSequence(data)
		if err != nil {
			return nil, fmt.Errorf("release %s: %w", tag, err)
		}
		sequences[stream] = sequence
	}
	return sequences, nil
}

// ensureTargetSequence adds a working-tree baseline only when no tag baseline exists.
// The target must have been validated by parseTargetStream.
func ensureTargetSequence(dir, target string, targetXy productstreams.XYVersion, sequences map[productstreams.XYVersion]int) error {
	if _, exists := sequences[targetXy]; exists {
		return nil
	}
	patch, _, _ := strings.Cut(strings.Split(target, ".")[2], "-")
	if patch != "0" && patch != "x" {
		return fmt.Errorf("missing initial release tag %s.0 for patch target %s", targetXy, target)
	}
	data, err := os.ReadFile(filepath.Join(dir, sequencePath))
	if err != nil {
		return fmt.Errorf("read current database sequence from %s: %w", sequencePath, err)
	}
	sequence, err := parseSequence(string(data))
	if err != nil {
		return fmt.Errorf("parse current database sequence from %s: %w", sequencePath, err)
	}
	sequences[targetXy] = sequence
	return nil
}

func validateReleaseSequences(target string, targetXy productstreams.XYVersion, sequences map[productstreams.XYVersion]int) ([]productstreams.XYVersion, error) {
	var err error
	floor := targetXy
	for range 3 {
		floor, err = productstreams.GetPreviousYStream(floor)
		if err != nil {
			return nil, fmt.Errorf("find minimum supported stream for %s: %w", target, err)
		}
	}
	for stream := floor; ; {
		// Every supported stream needs a baseline; RC-only streams count too.
		if _, exists := sequences[stream]; !exists {
			return nil, fmt.Errorf("missing initial release tag %s.0 required by %s; fetch release tags before generation", stream, target)
		}
		if stream == targetXy {
			break
		}
		stream = productstreams.GetNextYStream(stream)
	}
	streams := slices.Collect(maps.Keys(sequences))
	slices.SortFunc(streams, productstreams.XYVersion.Compare)
	last := 0
	for _, stream := range streams {
		sequence := sequences[stream]
		if sequence < last {
			return nil, fmt.Errorf("release %s sequence %d is less than previous sequence %d", stream, sequence, last)
		}
		last = sequence
	}
	return streams, nil
}

// renderReleaseVersions expects sorted streams returned by validateReleaseSequences.
func renderReleaseVersions(streams []productstreams.XYVersion, sequences map[productstreams.XYVersion]int) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by release-versions. DO NOT EDIT.\n\npackage migrations\n\nvar releaseVersions = []ReleaseVersion{")
	for _, stream := range streams {
		sequence := sequences[stream]
		fmt.Fprintf(&output, "{Version: %q, Sequence: %d},\n", stream.String(), sequence)
	}
	fmt.Fprintln(&output, "}")
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated release versions: %w", err)
	}
	return formatted, nil
}

func validateReleaseStream(stream productstreams.XYVersion) error {
	previous, err := productstreams.GetPreviousYStream(productstreams.GetNextYStream(stream))
	if err != nil {
		return err
	}
	if previous != stream {
		return fmt.Errorf("invalid release stream %s", stream)
	}
	return nil
}

// GA supersedes RCs; otherwise select the highest numeric RC ordinal.
func newerInitialRelease(candidate, selected string) bool {
	_, candidateRC, candidateIsRC := strings.Cut(candidate, "-rc.")
	_, selectedRC, selectedIsRC := strings.Cut(selected, "-rc.")
	if !candidateIsRC {
		return true
	}
	if !selectedIsRC {
		return false
	}
	// Ordinals are canonical non-negative decimal strings, so comparing lengths
	// avoids integer overflow while preserving numeric ordering.
	return len(candidateRC) > len(selectedRC) || (len(candidateRC) == len(selectedRC) && candidateRC > selectedRC)
}

func parseSequence(data string) (int, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "seq_num.go", data, 0)
	if err != nil {
		return 0, fmt.Errorf("parse seq_num.go: %w", err)
	}
	sequence := 0
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range decl.Names {
			if name.Name == "CurrentDBVersionSeqNum" && i < len(decl.Values) {
				if literal, ok := decl.Values[i].(*ast.BasicLit); ok && literal.Kind == token.INT {
					sequence, _ = strconv.Atoi(literal.Value)
				}
			}
		}
		return true
	})
	if sequence <= 0 {
		return 0, errors.New("missing positive CurrentDBVersionSeqNum in seq_num.go")
	}
	return sequence, nil
}

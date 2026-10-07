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

var initialRelease = regexp.MustCompile(`^[1-9][0-9]*\.(0|[1-9][0-9]*)\.0$`)
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
	if !targetVersion.MatchString(target) {
		return nil, fmt.Errorf("invalid target version %q", target)
	}
	xy, err := productstreams.ParseXYFromVersionString(target)
	if err != nil {
		return nil, fmt.Errorf("parse target stream from %q: %w", target, err)
	}
	previous, err := productstreams.GetPreviousYStream(productstreams.GetNextYStream(xy))
	if err != nil {
		return nil, fmt.Errorf("validate target stream %s: %w", xy, err)
	}
	if previous != xy {
		return nil, fmt.Errorf("invalid release stream %s", xy)
	}
	shallow, err := command(dir, "git", "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, fmt.Errorf("check whether checkout is shallow: %w", err)
	}
	if strings.TrimSpace(shallow) != "false" {
		return nil, errors.New("release generation requires a full checkout with GA tags")
	}
	tags, err := command(dir, "git", "tag", "--list")
	if err != nil {
		return nil, fmt.Errorf("list release tags: %w", err)
	}
	sequences := make(map[productstreams.XYVersion]int)
	var newestGA productstreams.XYVersion
	for tag := range strings.FieldsSeq(tags) {
		if !initialRelease.MatchString(tag) {
			continue
		}
		stream, err := productstreams.ParseXYFromVersionString(tag)
		if err != nil {
			return nil, fmt.Errorf("parse release stream from tag %q: %w", tag, err)
		}
		if stream.Compare(productstreams.XYVersion{X: 4, Y: 4}) < 0 || stream.Compare(xy) > 0 {
			continue
		}
		previous, err := productstreams.GetPreviousYStream(productstreams.GetNextYStream(stream))
		if err != nil {
			return nil, fmt.Errorf("validate historical release stream %s: %w", stream, err)
		}
		if previous != stream {
			return nil, fmt.Errorf("invalid historical release stream %s", stream)
		}
		data, err := command(dir, "git", "show", "refs/tags/"+tag+":"+sequencePath)
		if err != nil {
			return nil, fmt.Errorf("read %s from release tag %s: %w", sequencePath, tag, err)
		}
		sequence, err := parseSequence(data)
		if err != nil {
			return nil, fmt.Errorf("release %s: %w", tag, err)
		}
		sequences[stream] = sequence
		if stream.Compare(newestGA) > 0 {
			newestGA = stream
		}
	}
	if _, exists := sequences[xy]; !exists {
		patch, _, _ := strings.Cut(strings.Split(target, ".")[2], "-")
		if patch != "0" && patch != "x" {
			return nil, fmt.Errorf("missing initial GA tag %s.0 for patch target %s", xy, target)
		}
		data, err := os.ReadFile(filepath.Join(dir, sequencePath))
		if err != nil {
			return nil, fmt.Errorf("read current database sequence from %s: %w", sequencePath, err)
		}
		sequence, err := parseSequence(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse current database sequence from %s: %w", sequencePath, err)
		}
		sequences[xy] = sequence
	}
	floor := xy
	for range 3 {
		floor, err = productstreams.GetPreviousYStream(floor)
		if err != nil {
			return nil, fmt.Errorf("find minimum supported stream for %s: %w", target, err)
		}
	}
	for stream := floor; ; {
		// Pending streams after the newest GA have no historical baseline yet.
		if _, exists := sequences[stream]; !exists && (stream == floor || stream.Compare(newestGA) <= 0) {
			return nil, fmt.Errorf("missing initial GA tag %s.0 required by %s; fetch release tags before generation", stream, target)
		}
		if stream == xy {
			break
		}
		stream = productstreams.GetNextYStream(stream)
	}
	streams := slices.Collect(maps.Keys(sequences))
	slices.SortFunc(streams, productstreams.XYVersion.Compare)
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by release-versions. DO NOT EDIT.\n\npackage migrations\n\nvar releaseVersions = []ReleaseVersion{")
	last := 0
	for _, stream := range streams {
		sequence := sequences[stream]
		if sequence < last {
			return nil, fmt.Errorf("release %s sequence %d is less than previous sequence %d", stream, sequence, last)
		}
		last = sequence
		fmt.Fprintf(&output, "{Version: %q, Sequence: %d},\n", stream.String(), sequence)
	}
	fmt.Fprintln(&output, "}")
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated release versions: %w", err)
	}
	return formatted, nil
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

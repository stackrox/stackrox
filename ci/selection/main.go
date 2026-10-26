// Command selection classifies changed paths and gates enrolled Prow test work.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
)

type (
	// changedFiles binds a complete path list to immutable PR revisions.
	changedFiles struct {
		Base  string   `json:"base"`
		Head  string   `json:"head"`
		Files []string `json:"files"`
	}
	// pathGroup excludes paths only from this group, never from the diff.
	pathGroup struct {
		Include []string `json:"include"`
		Exclude []string `json:"exclude,omitempty"`
	}
	// fileFacts reports nonempty any/every matches; Ignored identifies demo filtering.
	fileFacts struct {
		Version  int      `json:"version"`
		Complete bool     `json:"complete"`
		Base     string   `json:"base"`
		Head     string   `json:"head"`
		Any      []string `json:"any"`
		Every    []string `json:"every"`
		Ignored  []string `json:"ignored,omitempty"`
	}
	// jobDecision permits existing execution unless an enrolled test explicitly skips.
	jobDecision struct {
		Version  int       `json:"version"`
		Job      string    `json:"job"`
		Decision string    `json:"decision"`
		Reason   string    `json:"reason"`
		Facts    fileFacts `json:"facts"`
	}
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil { //nolint:forbidigo // This command uses process JSON I/O.
		fmt.Fprintln(os.Stderr, err) //nolint:forbidigo // Diagnostics must not corrupt JSON stdout.
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("ci-selection", flag.ContinueOnError)
	groupsPath := flags.String("groups", "ci/selection/file-groups.json", "path-group configuration")
	ignorePath := flags.String("ignore", "", "demo-only path exclusions; omitted in production")
	job := flags.String("job", "", "normalized Prow test target; omit for filename facts")
	targetsPath := flags.String("targets", "ci/selection/prow-product-tests.json", "enrolled Prow product targets")
	base := flags.String("base", "", "immutable Git base revision for Prow")
	head := flags.String("head", "", "immutable Git PR head revision for Prow")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	var diff changedFiles
	if *base != "" || *head != "" {
		var err error
		diff, err = gitChanges(*base, *head)
		if err != nil {
			return err
		}
	} else if err := decode(input, &diff); err != nil {
		return fmt.Errorf("decode changed files: %w", err)
	}
	if err := validateDiff(diff); err != nil {
		return err
	}
	var groups map[string]pathGroup
	if err := readJSON(*groupsPath, &groups); err != nil {
		return err
	}
	var ignoredPatterns []string
	if *ignorePath != "" {
		if err := readJSON(*ignorePath, &ignoredPatterns); err != nil {
			return err
		}
	}
	facts, err := classify(diff, groups, ignoredPatterns)
	if err != nil {
		return err
	}
	var result any = facts
	if *job != "" {
		var targets []string
		if err := readJSON(*targetsPath, &targets); err != nil {
			return err
		}
		result = decide(*job, facts, targets)
	}
	if err := json.NewEncoder(output).Encode(result); err != nil {
		return fmt.Errorf("write selection: %w", err)
	}
	return nil
}

func decode(input io.Reader, result any) error {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected exactly one JSON value")
	}
	return nil
}

func readJSON(filename string, result any) error {
	encoded, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}
	if err := decode(bytes.NewReader(encoded), result); err != nil {
		return fmt.Errorf("read %s: %w", filename, err)
	}
	return nil
}

func validateDiff(diff changedFiles) error {
	validSHA := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if !validSHA.MatchString(diff.Base) || !validSHA.MatchString(diff.Head) || diff.Files == nil {
		return errors.New("complete diff requires full base/head SHAs and a files array")
	}
	for _, filename := range diff.Files {
		if filename == "" || strings.ContainsAny(filename, "\x00\r\n") || path.IsAbs(filename) || path.Clean(filename) != filename || filename == ".." || strings.HasPrefix(filename, "../") {
			return errors.New("invalid changed filename")
		}
	}
	return nil
}

// gitChanges compares the PR head with its merge base and preserves rename endpoints.
func gitChanges(base, head string) (changedFiles, error) {
	diff := changedFiles{Base: base, Head: head, Files: []string{}}
	if err := validateDiff(diff); err != nil {
		return changedFiles{}, err
	}
	mergeBase, err := exec.Command("git", "merge-base", "--all", base, head).Output()
	if err != nil {
		return changedFiles{}, fmt.Errorf("resolve merge base: %w", err)
	}
	ancestor := strings.TrimSpace(string(mergeBase))
	if strings.Contains(ancestor, "\n") || ancestor == "" {
		return changedFiles{}, errors.New("expected one merge base")
	}
	changed, err := exec.Command("git", "diff", "--no-ext-diff", "--no-renames", "--name-only", "-z", ancestor, head, "--").Output()
	if err != nil {
		return changedFiles{}, fmt.Errorf("collect changed paths: %w", err)
	}
	if len(changed) > 0 {
		if changed[len(changed)-1] != 0 {
			return changedFiles{}, errors.New("unterminated Git path list")
		}
		diff.Files = strings.Split(string(changed[:len(changed)-1]), "\x00")
	}
	if err := validateDiff(diff); err != nil {
		return changedFiles{}, err
	}
	return diff, nil
}

// classify validates all paths before applying optional demo filtering.
func classify(diff changedFiles, groups map[string]pathGroup, ignores []string) (fileFacts, error) {
	if err := validateDiff(diff); err != nil {
		return fileFacts{}, err
	}
	facts := fileFacts{Version: 1, Complete: true, Base: diff.Base, Head: diff.Head, Any: []string{}, Every: []string{}}
	if len(groups) == 0 {
		return fileFacts{}, errors.New("path groups must not be empty")
	}
	ignoreExpressions, err := compilePatterns(ignores)
	if err != nil {
		return fileFacts{}, fmt.Errorf("demo exclusions: %w", err)
	}
	files := make([]string, 0, len(diff.Files))
	for _, filename := range diff.Files {
		if matchesAny(ignoreExpressions, filename) {
			facts.Ignored = append(facts.Ignored, filename)
		} else {
			files = append(files, filename)
		}
	}
	for name, group := range groups {
		if name == "" || len(group.Include) == 0 {
			return fileFacts{}, errors.New("each group needs a name and inclusion pattern")
		}
		included, err := compilePatterns(group.Include)
		if err != nil {
			return fileFacts{}, fmt.Errorf("group %q includes: %w", name, err)
		}
		excluded, err := compilePatterns(group.Exclude)
		if err != nil {
			return fileFacts{}, fmt.Errorf("group %q excludes: %w", name, err)
		}
		anyMatch, everyMatch := false, len(files) > 0
		for _, filename := range files {
			matched := matchesAny(included, filename) && !matchesAny(excluded, filename)
			anyMatch = anyMatch || matched
			everyMatch = everyMatch && matched
		}
		if anyMatch {
			facts.Any = append(facts.Any, name)
		}
		if everyMatch {
			facts.Every = append(facts.Every, name)
		}
	}
	slices.Sort(facts.Any)
	slices.Sort(facts.Every)
	slices.Sort(facts.Ignored)
	return facts, nil
}

func matchesAny(expressions []*regexp.Regexp, filename string) bool {
	return slices.ContainsFunc(expressions, func(expression *regexp.Regexp) bool { return expression.MatchString(filename) })
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	expressions := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern == "" {
			return nil, errors.New("empty path pattern")
		}
		expression, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compile %q: %w", pattern, err)
		}
		expressions = append(expressions, expression)
	}
	return expressions, nil
}

func decide(job string, facts fileFacts, targets []string) jobDecision {
	result := jobDecision{Version: 1, Job: job, Decision: "run", Reason: "existing-policy", Facts: facts}
	if slices.Contains(targets, job) && facts.Version == 1 && facts.Complete && slices.Contains(facts.Every, "docs") {
		result.Decision, result.Reason = "skip", "docs-only"
	}
	return result
}

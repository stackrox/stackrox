// Command file-facts describes filename matches without selecting CI jobs.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
)

type (
	// changedFiles identifies the diff supplied by the existing file collector.
	changedFiles struct {
		Base  string   `json:"base"`
		Head  string   `json:"head"`
		Files []string `json:"files"`
	}

	// fileFacts reports nonempty any/every matches for named path groups.
	fileFacts struct {
		Version  int      `json:"version"`
		Complete bool     `json:"complete"`
		Base     string   `json:"base"`
		Head     string   `json:"head"`
		Any      []string `json:"any"`
		Every    []string `json:"every"`
	}
)

func main() {
	input, output, diagnostic := os.Stdin, os.Stdout, os.Stderr //nolint:forbidigo // This filter's interface is process JSON input and output.
	groupsPath := flag.String("groups", "ci/native-selectors/file-groups.json", "JSON object mapping path group names to Go regular expressions")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(diagnostic, "file-facts accepts only --groups; supply the collected diff as JSON on stdin")
		os.Exit(2)
	}
	if err := run(input, output, *groupsPath); err != nil {
		fmt.Fprintln(diagnostic, err)
		os.Exit(1)
	}
}

// run emits facts only after both the diff and all expressions validate.
func run(input io.Reader, output io.Writer, groupsPath string) error {
	var diff changedFiles
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&diff); err != nil {
		return fmt.Errorf("decode collected filenames: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected exactly one collected diff")
	}
	if diff.Base == "" || diff.Head == "" || diff.Files == nil {
		return errors.New("collected diff requires base, head, and a files array")
	}
	if slices.Contains(diff.Files, "") {
		return errors.New("filenames must not be empty")
	}
	encoded, err := os.ReadFile(groupsPath)
	if err != nil {
		return fmt.Errorf("read path groups: %w", err)
	}
	var groups map[string][]string
	if err := json.Unmarshal(encoded, &groups); err != nil {
		return fmt.Errorf("decode path groups: %w", err)
	}
	facts, err := classify(diff, groups)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(output).Encode(facts); err != nil {
		return fmt.Errorf("write file facts: %w", err)
	}
	return nil
}

// classify treats an empty diff as complete, with no positive every-file matches.
func classify(diff changedFiles, groups map[string][]string) (fileFacts, error) {
	facts := fileFacts{
		Version: 1, Complete: true, Base: diff.Base, Head: diff.Head,
		Any: []string{}, Every: []string{},
	}
	if len(groups) == 0 {
		return fileFacts{}, errors.New("path groups must not be empty")
	}
	for name, patterns := range groups {
		if name == "" || len(patterns) == 0 {
			return fileFacts{}, errors.New("each path group needs a name and at least one expression")
		}
		var expressions []*regexp.Regexp
		for _, pattern := range patterns {
			expression, err := regexp.Compile(pattern)
			if err != nil {
				return fileFacts{}, fmt.Errorf("path group %q: %w", name, err)
			}
			expressions = append(expressions, expression)
		}
		anyMatch, everyMatch := false, len(diff.Files) > 0
		for _, filename := range diff.Files {
			matched := slices.ContainsFunc(expressions, func(expression *regexp.Regexp) bool {
				return expression.MatchString(filename)
			})
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
	return facts, nil
}

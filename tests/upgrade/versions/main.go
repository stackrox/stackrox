// versions selects N-3 recovery and validates a pinned unsupported rollback release.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version/productstreams"
)

type release struct {
	Tag      string `json:"tag"`
	SHA      string `json:"sha"`
	Sequence int    `json:"sequence"`
}

type rollbackPlan struct {
	Target          string  `json:"target"`
	MinimumSequence int     `json:"minimum_sequence"`
	Allowed         release `json:"allowed"`
	Rejected        release `json:"rejected"`
}

func main() {
	target := flag.String("target", "", "current image version")
	from := flag.String("from", "", "last existing intermediate upgrade version")
	rejected := flag.String("rejected", "", "pinned unsupported GA release")
	flag.Parse()
	plan, err := makePlan(*target, *from, *rejected, migrations.MinimumSupportedForVersion, readRelease)
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(plan)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func makePlan(target, from, rejected string, minimum func(string) (migrations.ReleaseVersion, error), lookup func(string) (release, error)) (rollbackPlan, error) {
	plan := rollbackPlan{Target: target}
	floor, err := minimum(target)
	if err != nil {
		return plan, err
	}
	allowed, err := productstreams.ParseXYFromVersionString(floor.Version)
	if err != nil {
		return plan, err
	}
	last, err := productstreams.ParseXYFromVersionString(from)
	if err != nil {
		return plan, err
	}
	current, err := productstreams.ParseXYFromVersionString(target)
	if err != nil {
		return plan, err
	}
	if last.Compare(current) >= 0 {
		return plan, fmt.Errorf("intermediate release %s must precede target %s", from, target)
	}
	if last.Compare(allowed) < 0 {
		return plan, fmt.Errorf("intermediate release %s is below minimum supported stream %s; update PREVIOUS_RELEASES", from, allowed)
	}
	plan.MinimumSequence = floor.Sequence
	plan.Allowed, err = lookup(allowed.String() + ".0")
	if err != nil {
		return plan, err
	}
	plan.Rejected, err = lookup(rejected)
	if err != nil {
		return plan, err
	}
	if plan.Allowed.Sequence != floor.Sequence {
		return plan, fmt.Errorf("%s sequence %d disagrees with generated minimum %d", plan.Allowed.Tag, plan.Allowed.Sequence, floor.Sequence)
	}
	if plan.Rejected.Sequence >= floor.Sequence {
		return plan, fmt.Errorf("cannot require rollback rejection: %s supports sequence %d, minimum is %d; historical binaries enforce sequences, not release streams", plan.Rejected.Tag, plan.Rejected.Sequence, floor.Sequence)
	}
	return plan, nil
}

func readRelease(tag string) (release, error) {
	r := release{Tag: tag}
	ref := "refs/tags/" + tag
	sha, err := exec.Command("git", "rev-parse", "--verify", ref+"^{commit}").Output()
	if err != nil {
		return r, fmt.Errorf("missing GA tag %s: %w", tag, err)
	}
	r.SHA = strings.TrimSpace(string(sha))
	data, err := exec.Command("git", "show", r.SHA+":pkg/migrations/internal/seq_num.go").Output()
	if err != nil {
		return r, fmt.Errorf("reading sequence for %s: %w", tag, err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "seq_num.go", data, 0)
	if err != nil {
		return r, err
	}
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range decl.Names {
			if name.Name == "CurrentDBVersionSeqNum" && i < len(decl.Values) {
				if value, ok := decl.Values[i].(*ast.BasicLit); ok && value.Kind == token.INT {
					r.Sequence, _ = strconv.Atoi(value.Value)
				}
			}
		}
		return true
	})
	if r.Sequence <= 0 {
		return r, fmt.Errorf("invalid initial sequence in %s", tag)
	}
	return r, nil
}

#!/usr/bin/env python3
"""Exercise Prow's pinned matcher methods without running a controller or cluster."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile


def method(source, signature):
    """Extract a complete upstream Go method, preserving its implementation."""
    start = source.index(signature)
    end = source.index("\n}", start) + 2
    return source[start:end]


def main():
    """Compile upstream matching code with fixture types and execute its boundaries."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("jobs_go", type=Path, help="pkg/config/jobs.go from the pinned Prow commit")
    args = parser.parse_args()
    source = args.jobs_go.read_text()
    evidence = json.loads(Path(__file__).with_name("selectors.json").read_text())
    if hashlib.sha256(source.encode()).hexdigest() != evidence["matcher_sha256"]:
        raise ValueError("Prow source does not match the pinned experiment")
    extracted = "\n\n".join(method(source, signature) for signature in (
        "func (cm RegexpChangeMatcher) CouldRun(",
        "func (cm RegexpChangeMatcher) ShouldRun(",
        "func (cm RegexpChangeMatcher) RunsAgainstChanges(",
        "func (ps Presubmit) ShouldRun(",
    ))
    harness = r'''
package main
import ("errors"; "fmt"; "regexp")
type (
ChangedFilesProvider func() ([]string, error)
RegexpChangeMatcher struct {RunIfChanged, SkipIfOnlyChanged string; reChanges *regexp.Regexp}
Presubmit struct {AlwaysRun bool; RegexpChangeMatcher}
)
// CouldRun fixes the branch fixture; real branch matching is outside this experiment.
func (ps Presubmit) CouldRun(branch string) bool {return branch == "master"}
'''+extracted+r'''
func main() {
    selectors := map[string]Presubmit{
        "positive": {RegexpChangeMatcher: RegexpChangeMatcher{RunIfChanged:"^ui/", reChanges:regexp.MustCompile("^ui/")}},
        "negative": {RegexpChangeMatcher: RegexpChangeMatcher{SkipIfOnlyChanged:`^docs/|\.md$`, reChanges:regexp.MustCompile(`^docs/|\.md$`)}},
        "manual": {},
        "required": {AlwaysRun:true},
    }
    cases := map[string]struct {selector string; files []string; forced, defaults, unavailable, want, wantError bool}{
        "positive UI": {selector:"positive", files:[]string{"ui/App.tsx"}, want:true},
        "positive mixed diff": {selector:"positive", files:[]string{"ui/App.tsx","OWNERS"}, want:true},
        "negative docs": {selector:"negative", files:[]string{"README.md"}},
        "negative mixed diff": {selector:"negative", files:[]string{"README.md","central/a.go"}, want:true},
        "empty positive": {selector:"positive"},
        "empty negative": {selector:"negative"},
        "manual is dormant": {selector:"manual", files:[]string{"go.mod"}},
        "explicit test bypasses paths": {selector:"positive", files:[]string{"README.md"}, forced:true, want:true},
        "explicit test bypasses diff outage": {selector:"positive", forced:true, unavailable:true, want:true},
        "test all remains conditional": {selector:"positive", files:[]string{"README.md"}},
        "path API error": {selector:"positive", unavailable:true, wantError:true},
        "required job avoids path lookup": {selector:"required", unavailable:true, want:true},
    }
    for name, test := range cases {
        provider := func() ([]string,error) {if test.unavailable {return nil,errors.New("fixture failure")}; return test.files,nil}
        run, err := selectors[test.selector].ShouldRun("master", provider, test.forced, test.defaults)
        if run != test.want || (err != nil) != test.wantError {panic(fmt.Sprintf("%s: run=%t error=%v",name,run,err))}
    }
    fmt.Printf("%d upstream Prow matcher cases passed. Branch fixture only; no webhook, trust, or status reporting was executed.\n", len(cases))
}
'''
    with tempfile.TemporaryDirectory() as temp:
        file = Path(temp) / "main.go"
        file.write_text(harness)
        subprocess.run(["gofmt", "-w", str(file)], check=True)
        subprocess.run(["go", "run", str(file)], check=True)


if __name__ == "__main__":
    main()

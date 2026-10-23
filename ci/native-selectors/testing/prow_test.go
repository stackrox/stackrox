package testing_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestProwSelectors(t *testing.T) {
	f := newFixture(t)
	var evidence struct {
		Sources map[string]string
		Matcher struct {
			JobsGoSHA256 string `json:"jobs_go_sha256"`
		}
	}
	f.json(t, "prow/source.json", &evidence)
	dir := filepath.Join(t.TempDir(), "release")
	require.NoError(t, os.CopyFS(dir, os.DirFS(filepath.Join(f.root, "ci/native-selectors/prow/source"))))
	patch := filepath.Join(f.root, "ci/native-selectors/prow/native-selection.patch")
	command(t, dir, "", nil, "git", "apply", "--check", patch)
	command(t, dir, "", nil, "git", "apply", patch)
	var configured []object
	for file, hash := range evidence.Sources {
		before := f.read(t, "ci/native-selectors/prow/source/"+file)
		require.Equal(t, hash, digest(before))
		after, err := os.ReadFile(filepath.Join(dir, file))
		require.NoError(t, err)
		var old, updated object
		require.NoError(t, yaml.Unmarshal(before, &old))
		require.NoError(t, yaml.Unmarshal(after, &updated))
		for i, value := range array(old["tests"]) {
			a, b := obj(value), obj(array(updated["tests"])[i])
			if a["run_if_changed"] != b["run_if_changed"] || a["skip_if_only_changed"] != b["skip_if_only_changed"] {
				configured = append(configured, clone(t, b))
				for _, key := range []string{"run_if_changed", "skip_if_only_changed"} {
					if value, ok := a[key]; ok {
						b[key] = value
					} else {
						delete(b, key)
					}
				}
			}
		}
		require.Equal(t, old, updated, "only path selectors may change in %s", file)
	}
	require.Len(t, configured, 4)
	var vm []object
	var qa, nongroovy object
	for _, job := range configured {
		switch job["as"] {
		case "vm-scanning-e2e-tests":
			vm = append(vm, job)
		case "gke-qa-e2e-tests":
			qa = job
		case "gke-nongroovy-e2e-tests":
			nongroovy = job
		}
	}
	require.Len(t, vm, 2)
	require.NotEmpty(t, qa)
	require.NotEmpty(t, nongroovy)
	for name, test := range map[string]struct {
		paths []string
		match bool
	}{
		"roxagent":                      {[]string{"compliance/virtualmachines/roxagent/main.go"}, true},
		"VM mixed with unrelated files": {[]string{"sensor/common/virtualmachine/vmscraper/scraper.go", "ui/package.json", "README.md"}, true},
		"Central conversion":            {[]string{"central/convert/internaltostorage/virtual_machine_v2.go"}, true},
		"shared concurrency":            {[]string{"pkg/concurrency/context.go"}, true},
		"Scanner shared implementation": {[]string{"scanner/indexer/repositorytocpefetcher.go"}, true},
		"operator":                      {[]string{"operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml"}, true},
		"VM CI entry":                   {[]string{"scripts/ci/jobs/ocp_vm_scanning_e2e_tests.py"}, true},
		"roxctl":                        {[]string{"roxctl/main.go"}, true}, "roxie version": {[]string{"ROXIE_VERSION"}, true}, "roxie wrapper": {[]string{"scripts/roxie.sh"}, true},
		"suite": {[]string{"tests/vm_scanning_suite_test.go"}, true}, "Go dependencies": {[]string{"go.mod"}, true},
		"proto outside known directories": {[]string{"elsewhere/schema.proto"}, true}, "conservative Central envelope": {[]string{"central/reports/README.md"}, true},
		"UI and docs": {[]string{"ui/package.json", "README.md"}, false}, "irrelevant": {[]string{"README.md", "OWNERS"}, false},
		"VM name alone": {[]string{"unrelated/vm-name.txt"}, false}, "empty": {nil, false},
	} {
		t.Run(name, func(t *testing.T) {
			for _, job := range vm {
				require.Equal(t, false, job["always_run"])
				require.Equal(t, true, job["optional"])
				require.Equal(t, test.match, pathMatch(job, test.paths))
			}
		})
	}
	require.True(t, pathMatch(qa, []string{"go.mod", "README.md"}))
	require.False(t, pathMatch(qa, []string{"README.md"}))
	require.True(t, pathMatch(nongroovy, []string{"elsewhere/schema.proto", "README.md"}))
	require.False(t, pathMatch(nongroovy, []string{"roxctl/main.go"}), "Prow nongroovy has no positive roxctl rule in the comparison")
	master := f.yaml(t, "ci/native-selectors/prow/source/ci-operator/config/stackrox/stackrox/stackrox-stackrox-master.yaml")
	uiFound := false
	for _, job := range array(master["tests"]) {
		if obj(job)["as"] == "gke-ui-e2e-tests" {
			require.Equal(t, true, obj(job)["always_run"])
			uiFound = true
		}
	}
	require.True(t, uiFound)
	t.Run("pinned upstream matcher", func(t *testing.T) {
		file := os.Getenv("NATIVE_PROW_JOBS_GO")
		if file == "" {
			t.Skip("set NATIVE_PROW_JOBS_GO to execute the pinned upstream matcher methods")
		}
		source, err := os.ReadFile(file)
		require.NoError(t, err)
		require.Equal(t, evidence.Matcher.JobsGoSHA256, digest(source))
		var methods []string
		for _, signature := range []string{
			"func (cm RegexpChangeMatcher) CouldRun(", "func (cm RegexpChangeMatcher) ShouldRun(", "func (cm RegexpChangeMatcher) RunsAgainstChanges(",
			"func (ps Presubmit) CouldRun(", "func (ps Presubmit) ShouldRun(",
		} {
			start := strings.Index(string(source), signature)
			require.GreaterOrEqual(t, start, 0)
			end := strings.Index(string(source[start:]), "\n}")
			require.GreaterOrEqual(t, end, 0)
			methods = append(methods, string(source[start:start+end+2]))
		}
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module native-prow-matcher\n\ngo 1.26.7\n"), 0o600))
		harness := strings.ReplaceAll(prowHarness, "VM_PATTERN", strconv.Quote(str(vm[0]["run_if_changed"])))
		harness = strings.ReplaceAll(harness, "BROAD_PATTERN", strconv.Quote(str(qa["run_if_changed"])))
		packageStart := strings.Index(string(source), "package config")
		require.GreaterOrEqual(t, packageStart, 0)
		generated := filepath.Join(dir, "matcher_test.go")
		require.NoError(t, os.WriteFile(generated, []byte(string(source[:packageStart])+harness+"\n"+strings.Join(methods, "\n\n")), 0o600))
		command(t, dir, "", nil, "gofmt", "-w", generated)
		t.Log(command(t, dir, "", nil, "go", "test", "-count=1", "-v", "./..."))
	})
}

func pathMatch(job object, paths []string) bool {
	pattern := str(job["run_if_changed"])
	if pattern == "" {
		return false
	}
	expression := regexp.MustCompile(pattern)
	return slices.ContainsFunc(paths, expression.MatchString)
}

const prowHarness = `
package config
import ("errors"; "regexp"; "testing")
// ChangedFilesProvider supplies the fixture's changed files.
type ChangedFilesProvider func() ([]string, error)
// RegexpChangeMatcher holds native selector fields used by the upstream methods.
type RegexpChangeMatcher struct {RunIfChanged, SkipIfOnlyChanged string; reChanges *regexp.Regexp}
// Brancher fixes branch eligibility while the real path matcher runs.
type Brancher struct{}
// ShouldRun permits the fixture's master branch.
func (Brancher) ShouldRun(string) bool {return true}
// Presubmit supplies the fields required by the upstream methods.
type Presubmit struct {Brancher; RegexpChangeMatcher; AlwaysRun bool}
func TestNativeMatcher(t *testing.T) {
 tests:=map[string]struct{pattern string; skip, always, force, fetchError bool; files []string; want, wantError bool}{
  "VM feature mixed with unrelated files":{pattern:VM_PATTERN,files:[]string{"sensor/common/virtualmachine/vmscraper/scraper.go","README.md"},want:true},
  "shared dependency":{pattern:VM_PATTERN,files:[]string{"pkg/concurrency/context.go"},want:true},
  "irrelevant paths":{pattern:VM_PATTERN,files:[]string{"README.md"}},
  "empty diff":{pattern:VM_PATTERN},
  "explicit test bypasses paths":{pattern:VM_PATTERN,force:true,files:[]string{"README.md"},want:true},
  "explicit test bypasses fetch failure":{pattern:VM_PATTERN,force:true,fetchError:true,want:true},
  "conditional fetch failure propagates":{pattern:VM_PATTERN,fetchError:true,wantError:true},
  "manual job stays dormant":{files:[]string{"go.mod"}},
  "manual job explicit request":{force:true,want:true},
  "test all remains conditional for docs":{pattern:BROAD_PATTERN,files:[]string{"README.md"}},
  "test all matches broad dependencies":{pattern:BROAD_PATTERN,files:[]string{"go.mod","README.md"},want:true},
  "required always-run UI ignores diff failure":{always:true,fetchError:true,want:true},
  "any UI selects UI alone":{pattern:"^ui/",files:[]string{"ui/package.json"},want:true},
  "any UI also selects a mixed diff":{pattern:"^ui/",files:[]string{"ui/package.json","OWNERS"},want:true},
  "skip-only UI excludes UI alone":{pattern:"^ui/",skip:true,files:[]string{"ui/package.json"}},
  "skip-only UI permits a mixed diff":{pattern:"^ui/",skip:true,files:[]string{"ui/package.json","OWNERS"},want:true},
 }
 for name,test:=range tests {t.Run(name,func(t *testing.T){
  matcher:=RegexpChangeMatcher{}
  if test.pattern!="" {matcher.reChanges=regexp.MustCompile(test.pattern);if test.skip {matcher.SkipIfOnlyChanged=test.pattern}else{matcher.RunIfChanged=test.pattern}}
  job:=Presubmit{RegexpChangeMatcher:matcher,AlwaysRun:test.always}
  got,err:=job.ShouldRun("master",func()([]string,error){if test.fetchError{return nil,errors.New("unavailable")};return test.files,nil},test.force,false)
  if got!=test.want||(err!=nil)!=test.wantError {t.Fatalf("run=%v error=%v; want run=%v error=%v",got,err,test.want,test.wantError)}
 })}
}
`

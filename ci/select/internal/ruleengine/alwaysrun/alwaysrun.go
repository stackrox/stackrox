// Package alwaysrun is a placeholder engine that runs every target. It
// reads the target inventory from the rules file and plans "run" for each,
// ignoring the changed paths, filling the engine seam until a real
// path-based engine replaces it.
package alwaysrun

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/ruleengine"
	"go.yaml.in/yaml/v3"
)

// New returns an engine that runs every target in the rules file.
func New() ruleengine.Engine { return alwaysRun{} }

type alwaysRun struct{}

// rulesDoc is the minimal rules shape this stub reads: just the target
// inventory. A real engine will add path rules alongside it.
type rulesDoc struct {
	Version int      `yaml:"version"`
	Jobs    []string `yaml:"jobs"`
}

func (alwaysRun) Plan(rules []byte, changed []string) (ruleengine.Decision, error) {
	var doc rulesDoc
	if err := yaml.Unmarshal(rules, &doc); err != nil {
		return ruleengine.Decision{}, errors.Wrap(err, "parse rules")
	}
	if len(doc.Jobs) == 0 {
		return ruleengine.Decision{}, errors.New("rules list no jobs")
	}
	actions := make(map[string]string, len(doc.Jobs))
	for _, job := range doc.Jobs {
		actions[job] = "run"
	}
	log := fmt.Sprintf("always-run engine: %d changed path(s); every target runs\n", len(changed))
	return ruleengine.Decision{Actions: actions, Log: log}, nil
}

// Package model implements the learned-edge sensor: dependency edges
// between input files and output files, inferred from co-change
// history on full builds, validated by shadow/audit runs.
//
// Safety rules (see SENSORS.md):
//   - Only POSITIVE edges are learned: (input changed, output changed)
//     in the same full build. "Output did not change" is never recorded
//     as an absence — a comment-only input change leaves outputs
//     byte-identical without making them independent.
//   - Only FULL builds produce evidence. A build whose outputs were
//     partially restored from cache validates nothing (circularity).
//   - Predictions are advisory until an edge survives validation.
package model

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
)

// Edge identifies a dependency: changes to Input co-occurred with
// changes to Output.
type Edge struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// Evidence is what we know about an edge.
type Evidence struct {
	CoChanges int       `json:"co_changes"` // full builds where both changed
	Missed    int       `json:"missed"`     // validations where the edge was absent from a prediction but needed
	Validated bool      `json:"validated"`  // survived a shadow/audit scoring
	LastSeen  time.Time `json:"last_seen"`
}

// BuildRecord is one observed build.
type BuildRecord struct {
	Time    time.Time         `json:"time"`
	Command string            `json:"command"`
	Full    bool              `json:"full"`              // outputs produced, not restored
	Shadow  bool              `json:"shadow,omitempty"`  // predictions were scored, not trusted
	Inputs  map[string]string `json:"inputs,omitempty"`  // path → hash
	Outputs map[string]string `json:"outputs,omitempty"` // path → hash
}

// Model is the learned dependency graph for one codebase.
type Model struct {
	Path   string
	Edges  map[Edge]*Evidence
	Builds []BuildRecord
}

// modelJSON mirrors Model with string edge keys (JSON object keys
// must be strings; Edge is a struct).
type modelJSON struct {
	Edges  map[string]*Evidence `json:"edges"`
	Builds []BuildRecord        `json:"builds"`
}

func edgeKey(e Edge) string { return e.Input + "\x00" + e.Output }

func (m *Model) MarshalJSON() ([]byte, error) {
	edges := make(map[string]*Evidence, len(m.Edges))
	for e, v := range m.Edges {
		edges[edgeKey(e)] = v
	}
	return json.Marshal(modelJSON{Edges: edges, Builds: m.Builds})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var mj modelJSON
	if err := json.Unmarshal(data, &mj); err != nil {
		return err
	}
	m.Edges = make(map[Edge]*Evidence, len(mj.Edges))
	for k, v := range mj.Edges {
		i := strings.IndexByte(k, 0)
		if i < 0 {
			continue
		}
		m.Edges[Edge{k[:i], k[i+1:]}] = v
	}
	m.Builds = mj.Builds
	return nil
}

// Load reads the model from path (empty model if absent).
func Load(path string) (*Model, error) {
	m := &Model{Path: path, Edges: make(map[Edge]*Evidence)}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	if m.Edges == nil {
		m.Edges = make(map[Edge]*Evidence)
	}
	return m, nil
}

// Save writes the model back to disk.
func (m *Model) Save() error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0o644)
}

// LastFullBuild returns the most recent full build record, or nil.
func (m *Model) LastFullBuild() *BuildRecord {
	for i := len(m.Builds) - 1; i >= 0; i-- {
		if m.Builds[i].Full {
			return &m.Builds[i]
		}
	}
	return nil
}

// RecordBuild appends a build record and, when it is a full build
// following an earlier full build, learns edges from co-changes:
// every (changed input, changed output) pair gains evidence.
func (m *Model) RecordBuild(rec BuildRecord) {
	if rec.Full {
		if prev := m.LastFullBuild(); prev != nil {
			changedIn := diffMaps(prev.Inputs, rec.Inputs)
			changedOut := diffMaps(prev.Outputs, rec.Outputs)
			now := time.Now()
			for _, in := range changedIn {
				for _, out := range changedOut {
					e := m.Edges[Edge{in, out}]
					if e == nil {
						e = &Evidence{}
						m.Edges[Edge{in, out}] = e
					}
					e.CoChanges++
					e.LastSeen = now
				}
			}
		}
	}
	m.Builds = append(m.Builds, rec)
	// Bound history: keep the most recent 100 builds.
	if len(m.Builds) > 100 {
		m.Builds = m.Builds[len(m.Builds)-100:]
	}
}

// ChangedSince returns the input paths that differ from the last full
// build (added, removed, or content-changed).
func (m *Model) ChangedSince(rec BuildRecord) []string {
	prev := m.LastFullBuild()
	if prev == nil {
		return nil
	}
	return diffMaps(prev.Inputs, rec.Inputs)
}

// Predict returns the outputs the model believes the changed inputs
// affect. conservative=true means at least one changed file has no
// positive edges at all — the caller must assume everything is
// affected (rule: unknown → full rebuild).
func (m *Model) Predict(changed []string) (affected map[string]bool, conservative bool) {
	affected = make(map[string]bool)
	for _, f := range changed {
		found := false
		for e := range m.Edges {
			if e.Input == f {
				affected[e.Output] = true
				found = true
			}
		}
		if !found {
			conservative = true
		}
	}
	return affected, conservative
}

// ScoreReport is the outcome of comparing a prediction to ground truth.
type ScoreReport struct {
	Predicted   []string // outputs the prediction marked affected
	Actually    []string // outputs that really changed
	MissedEdges []Edge   // edges the model lacked (ground truth won)
	Spurious    int      // predicted-affected but did not change (imprecision, not error)
	Recall      float64
}

// Score compares the model's prediction for changed inputs against the
// set of outputs that actually changed in a full build. Missed edges
// are added to the model (repair) and everything exercised is marked
// validated.
func (m *Model) Score(changed []string, actualChanged []string) ScoreReport {
	predicted, _ := m.Predict(changed)
	actual := make(map[string]bool, len(actualChanged))
	for _, o := range actualChanged {
		actual[o] = true
	}

	rep := ScoreReport{Actually: actualChanged}
	for o := range predicted {
		rep.Predicted = append(rep.Predicted, o)
	}
	sort.Strings(rep.Predicted)

	now := time.Now()
	for _, o := range actualChanged {
		if !predicted[o] {
			// Ground truth says this output was affected; the model
			// missed at least one edge into it. Attribute to every
			// changed input (over-attribution is safe: it only
			// over-invalidates).
			for _, f := range changed {
				e := Edge{f, o}
				ev := m.Edges[e]
				if ev == nil {
					ev = &Evidence{}
					m.Edges[e] = ev
				}
				ev.Missed++
				ev.LastSeen = now
				rep.MissedEdges = append(rep.MissedEdges, e)
			}
		}
	}
	for o := range predicted {
		if !actual[o] {
			rep.Spurious++
		}
	}

	if len(actualChanged) > 0 {
		rep.Recall = 1 - float64(countMissedOutputs(predicted, actualChanged))/float64(len(actualChanged))
	}

	// Everything that was exercised correctly gets validated.
	for e, ev := range m.Edges {
		if predicted[e.Output] && actual[e.Output] {
			ev.Validated = true
		}
	}
	return rep
}

func countMissedOutputs(predicted map[string]bool, actual []string) int {
	var n int
	for _, o := range actual {
		if !predicted[o] {
			n++
		}
	}
	return n
}

// diffMaps returns keys whose value differs between a and b, plus keys
// added or removed.
func diffMaps(a, b map[string]string) []string {
	var out []string
	for k, v := range b {
		if av, ok := a[k]; !ok || av != v {
			out = append(out, k)
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

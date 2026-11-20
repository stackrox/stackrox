package model

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestModel(t *testing.T) *Model {
	return &Model{Path: filepath.Join(t.TempDir(), "model.json"), Edges: make(map[Edge]*Evidence)}
}

func TestLearnPositiveEdgesOnly(t *testing.T) {
	m := newTestModel(t)

	// Build 1: baseline.
	m.RecordBuild(BuildRecord{
		Full: true, Inputs: map[string]string{"a.go": "1", "b.go": "1"},
		Outputs: map[string]string{"bin/x": "1", "bin/y": "1"},
	})
	// Build 2: a.go changed, bin/x changed, bin/y did NOT change.
	m.RecordBuild(BuildRecord{
		Full: true, Inputs: map[string]string{"a.go": "2", "b.go": "1"},
		Outputs: map[string]string{"bin/x": "2", "bin/y": "1"},
	})

	// Positive edge learned.
	if e := m.Edges[Edge{"a.go", "bin/x"}]; e == nil || e.CoChanges != 1 {
		t.Fatalf("expected edge a.go→bin/x with 1 co-change, got %+v", e)
	}
	// NO negative edge: a.go changed while bin/y did not — must NOT
	// be recorded as independence (comment-change trap).
	if len(m.Edges) != 1 {
		t.Fatalf("expected exactly 1 edge, got %d: %v", len(m.Edges), m.Edges)
	}
}

func TestPartialBuildsProduceNoEvidence(t *testing.T) {
	m := newTestModel(t)
	m.RecordBuild(BuildRecord{Full: true,
		Inputs: map[string]string{"a.go": "1"}, Outputs: map[string]string{"bin/x": "1"}})
	// A cache-hit build (Full=false) must not learn even if we recorded it.
	m.RecordBuild(BuildRecord{Full: false,
		Inputs: map[string]string{"a.go": "2"}, Outputs: map[string]string{"bin/x": "1"}})
	if len(m.Edges) != 0 {
		t.Fatalf("non-full build must not produce evidence, got %d edges", len(m.Edges))
	}
}

func TestPredictConservativeOnUnknown(t *testing.T) {
	m := newTestModel(t)
	m.Edges[Edge{"a.go", "bin/x"}] = &Evidence{CoChanges: 3, Validated: true}

	affected, conservative := m.Predict([]string{"a.go"})
	if conservative {
		t.Fatal("known file should not be conservative")
	}
	if !affected["bin/x"] {
		t.Fatal("expected bin/x affected")
	}

	_, conservative = m.Predict([]string{"a.go", "new.go"})
	if !conservative {
		t.Fatal("unknown file must be conservative")
	}
}

func TestScoreDetectsMissedEdges(t *testing.T) {
	m := newTestModel(t)
	// Model thinks only b.go→bin/x exists.
	m.Edges[Edge{"b.go", "bin/x"}] = &Evidence{CoChanges: 2}

	// Ground truth: this time only a.go changed, and bin/x changed —
	// unambiguous attribution: the model lacked a.go→bin/x.
	rep := m.Score([]string{"a.go"}, []string{"bin/x"})

	if len(rep.MissedEdges) == 0 {
		t.Fatal("expected missed edge a.go→bin/x")
	}
	if e := m.Edges[Edge{"a.go", "bin/x"}]; e == nil || e.Missed != 1 {
		t.Fatalf("missed edge should be repaired into model, got %+v", e)
	}
	// Attribution is ambiguous when both inputs change — a.go→bin/x
	// is NOT learned here (correct conservatism). Learn it via a
	// second run where only a.go changes:
	m.Score([]string{"a.go"}, []string{"bin/x"})
	if e := m.Edges[Edge{"a.go", "bin/x"}]; e == nil {
		t.Fatal("a.go→bin/x should be repaired into the model")
	}

	// A run where only b.go changes exercises the known edge and
	// validates it.
	m.Score([]string{"b.go"}, []string{"bin/x"})
	if !m.Edges[Edge{"b.go", "bin/x"}].Validated {
		t.Fatal("exercised correct edge should be validated")
	}
	if rep.Recall != 0 {
		t.Fatalf("recall 0 expected (bin/x not predicted from a.go alone), got %v", rep.Recall)
	}
}

func TestScoreSpuriousNotError(t *testing.T) {
	m := newTestModel(t)
	m.Edges[Edge{"a.go", "bin/old"}] = &Evidence{CoChanges: 1}

	rep := m.Score([]string{"a.go"}, []string{"bin/new"})
	if rep.Spurious != 1 {
		t.Fatalf("expected 1 spurious prediction, got %d", rep.Spurious)
	}
	// Spurious is imprecision, not error — must not fail validation
	// of other edges; and no crash.
	if rep.Recall != 0 {
		t.Fatalf("recall 0 expected (bin/new missed), got %v", rep.Recall)
	}
}

func TestModelPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.json")
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	m.RecordBuild(BuildRecord{Full: true,
		Inputs: map[string]string{"a.go": "1"}, Outputs: map[string]string{"bin/x": "1"}})
	m.RecordBuild(BuildRecord{Full: true,
		Inputs: map[string]string{"a.go": "2"}, Outputs: map[string]string{"bin/x": "2"}})
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	m2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Edges[Edge{"a.go", "bin/x"}] == nil {
		t.Fatal("edge lost across save/load")
	}
	if len(m2.Builds) != 2 {
		t.Fatalf("expected 2 build records, got %d", len(m2.Builds))
	}
}

func TestChangedSince(t *testing.T) {
	m := newTestModel(t)
	m.RecordBuild(BuildRecord{Full: true,
		Inputs: map[string]string{"a.go": "1", "b.go": "1"}, Outputs: nil})
	changed := m.ChangedSince(BuildRecord{Inputs: map[string]string{"a.go": "1", "b.go": "2", "c.go": "1"}})
	if len(changed) != 2 {
		t.Fatalf("expected b.go (changed) + c.go (new), got %v", changed)
	}
}

func TestLoadMissingFile(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Edges == nil || len(m.Edges) != 0 {
		t.Fatal("missing file should give empty model")
	}
	if _, err := os.Stat(m.Path); !os.IsNotExist(err) {
		t.Fatal("load should not create the file")
	}
}

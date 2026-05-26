package benchmark

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	scenarioFileName     = "scenario.yaml"
	phaseWaitInitialSync = "wait_initial_sync"
	phaseSteady          = "steady"
	defaultMaturity      = "dev"
)

// Scenario describes a sensor benchmark scenario loaded from scenario.yaml.
type Scenario struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   ScenarioMetadata `yaml:"metadata"`
	Spec       ScenarioSpec     `yaml:"spec"`

	scenarioDir      string
	resolvedWorkload string
	resolvedPolicies string
}

// ScenarioMetadata holds scenario identity and labels.
type ScenarioMetadata struct {
	Name    string            `yaml:"name"`
	Version string            `yaml:"version"`
	Labels  map[string]string `yaml:"labels"`
}

// ScenarioSpec holds workload paths, sensor settings, and benchmark phases.
type ScenarioSpec struct {
	Workload string       `yaml:"workload"`
	Policies string       `yaml:"policies"`
	Sensor   SensorConfig `yaml:"sensor"`
	Phases   []Phase      `yaml:"phases"`
}

// SensorConfig holds in-process sensor harness settings.
type SensorConfig struct {
	MetricsPort string `yaml:"metricsPort"`
}

// Phase is one step in the benchmark lifecycle (e.g. sync wait, steady measurement).
type Phase struct {
	Name     string `yaml:"name"`
	MaxWait  string `yaml:"maxWait,omitempty"`
	Duration string `yaml:"duration,omitempty"`
}

// LoadScenario reads scenario.yaml from dir and resolves workload and policy paths.
// dir may be relative to the current working directory or, if not found there,
// relative to the repository root (directory containing go.mod).
func LoadScenario(dir string) (*Scenario, error) {
	resolvedDir, err := resolveScenarioDir(dir)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(resolvedDir, scenarioFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "reading scenario file %q", path)
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, errors.Wrapf(err, "parsing scenario file %q", path)
	}

	s.scenarioDir = resolvedDir
	s.resolvedWorkload = resolvePath(resolvedDir, s.Spec.Workload)
	s.resolvedPolicies = resolvePath(resolvedDir, s.Spec.Policies)

	return &s, nil
}

func resolveScenarioDir(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	if scenarioExists(dir) {
		return dir, nil
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(root, dir)
	if scenarioExists(candidate) {
		return candidate, nil
	}
	return "", errors.Errorf("scenario directory %q not found", dir)
}

func scenarioExists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, scenarioFileName))
	return err == nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "getting working directory")
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found")
		}
		dir = parent
	}
}

func resolvePath(dir, rel string) string {
	return filepath.Clean(filepath.Join(dir, rel))
}

// SteadyDuration returns the duration of the steady measurement phase.
func (s *Scenario) SteadyDuration() time.Duration {
	return s.phaseDuration(phaseSteady)
}

// MaxSyncWait returns the maximum wait for initial sync before failing.
func (s *Scenario) MaxSyncWait() time.Duration {
	return s.phaseMaxWait(phaseWaitInitialSync)
}

// ResolvedWorkloadPath returns the workload file path relative to the process cwd.
func (s *Scenario) ResolvedWorkloadPath() string {
	return s.resolvedWorkload
}

// ResolvedPoliciesPath returns the policies file path relative to the process cwd.
func (s *Scenario) ResolvedPoliciesPath() string {
	return s.resolvedPolicies
}

// Maturity returns metadata.labels.maturity, defaulting to "dev".
func (s *Scenario) Maturity() string {
	if s.Metadata.Labels != nil {
		if m, ok := s.Metadata.Labels["maturity"]; ok && m != "" {
			return m
		}
	}
	return defaultMaturity
}

// MetricsPort returns the sensor metrics listen address from the scenario spec.
func (s *Scenario) MetricsPort() string {
	return s.Spec.Sensor.MetricsPort
}

func (s *Scenario) phaseDuration(name string) time.Duration {
	for _, phase := range s.Spec.Phases {
		if phase.Name != name || phase.Duration == "" {
			continue
		}
		d, err := time.ParseDuration(phase.Duration)
		if err != nil {
			return 0
		}
		return d
	}
	return 0
}

func (s *Scenario) phaseMaxWait(name string) time.Duration {
	for _, phase := range s.Spec.Phases {
		if phase.Name != name || phase.MaxWait == "" {
			continue
		}
		d, err := time.ParseDuration(phase.MaxWait)
		if err != nil {
			return 0
		}
		return d
	}
	return 0
}

package main

import (
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := map[string]struct {
		config    *Config
		wantError bool
		errorMsg  string
	}{
		"valid config": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go", "style-check"},
				Defaults: map[string]string{
					"go":          "run",
					"style-check": "run",
				},
				Rules: []Rule{
					{
						Name: "go-changes",
						When: Condition{
							AnyFileMatches: []string{`\.go$`},
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: false,
		},
		"unsupported version": {
			config: &Config{
				Version: 99,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
			},
			wantError: true,
			errorMsg:  "unsupported config version: 99",
		},
		"no jobs defined": {
			config: &Config{
				Version:  1,
				Jobs:     []string{},
				Defaults: map[string]string{},
			},
			wantError: true,
			errorMsg:  "no jobs defined",
		},
		"job missing default": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go", "ui"},
				Defaults: map[string]string{
					"go": "run",
					// ui missing
				},
			},
			wantError: true,
			errorMsg:  `job "ui" has no default`,
		},
		"invalid default value": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "maybe", // invalid
				},
			},
			wantError: true,
			errorMsg:  `has invalid default "maybe"`,
		},
		"rule with no name": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						// Name missing
						When: Condition{
							AnyFileMatches: []string{`\.go$`},
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "rule 0 has no name",
		},
		"rule with no condition": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "empty-rule",
						When: Condition{
							// No conditions
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `rule "empty-rule" has no condition`,
		},
		"rule with multiple conditions": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "multi-condition",
						When: Condition{
							Label:          "ci-run-all",
							AnyFileMatches: []string{`\.go$`}, // Both set - invalid
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `rule "multi-condition" has multiple conditions`,
		},
		"rule with no action": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "no-action",
						When: Condition{
							AnyFileMatches: []string{`\.go$`},
						},
						Action: Action{
							// No run or skip
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `rule "no-action" has no action`,
		},
		"every_file_matches condition": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "skip",
				},
				Rules: []Rule{
					{
						Name: "docs-only",
						When: Condition{
							EveryFileMatches: []string{`\.md$`, `^docs/`},
						},
						Action: Action{
							Skip: []string{"go"},
						},
					},
				},
			},
			wantError: false,
		},
		"any_file_changed condition": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"style-check"},
				Defaults: map[string]string{
					"style-check": "run",
				},
				Rules: []Rule{
					{
						Name: "always-style",
						When: Condition{
							AnyFileChanged: true,
						},
						Action: Action{
							Run: []string{"style-check"},
						},
					},
				},
			},
			wantError: false,
		},
		"duplicate job name": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go", "style-check", "go"}, // duplicate "go"
				Defaults: map[string]string{
					"go":          "run",
					"style-check": "run",
				},
			},
			wantError: true,
			errorMsg:  `duplicate job name "go"`,
		},
		"duplicate rule name": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "go-changes",
						When: Condition{
							AnyFileMatches: []string{`\.go$`},
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
					{
						Name: "go-changes", // duplicate
						When: Condition{
							Label: "ci-run-all",
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `duplicate rule name "go-changes"`,
		},
		"invalid regex pattern in any_file_matches": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "bad-regex",
						When: Condition{
							AnyFileMatches: []string{`[invalid`}, // unclosed bracket
						},
						Action: Action{
							Run: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `rule "bad-regex" has invalid regex pattern`,
		},
		"invalid regex pattern in every_file_matches": {
			config: &Config{
				Version: 1,
				Jobs:    []string{"go"},
				Defaults: map[string]string{
					"go": "run",
				},
				Rules: []Rule{
					{
						Name: "bad-regex-every",
						When: Condition{
							EveryFileMatches: []string{`*invalid*`}, // invalid quantifier
						},
						Action: Action{
							Skip: []string{"go"},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  `rule "bad-regex-every" has invalid regex pattern`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorMsg)
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package mitre

import (
	"encoding/json"
	"testing"
)

// FuzzUnmarshalMitreAttackBundle tests that UnmarshalAndExtractMitreAttackBundle
// never panics on arbitrary input, including malformed JSON and edge cases.
func FuzzUnmarshalMitreAttackBundle(f *testing.F) {
	// Seed with minimal valid JSON that matches the expected structure
	f.Add([]byte(`{"type":"bundle","objects":[]}`))

	// Seed with a more realistic minimal bundle with metadata
	f.Add([]byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "x-mitre-collection",
				"x_mitre_version": "9.0"
			}
		]
	}`))

	// Seed with a minimal tactic
	f.Add([]byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "x-mitre-tactic",
				"name": "Test Tactic",
				"description": "Test description",
				"x_mitre_domains": ["enterprise-attack"],
				"x_mitre_shortname": "test",
				"external_references": [
					{
						"source_name": "mitre-attack",
						"external_id": "TA0001"
					}
				]
			}
		]
	}`))

	// Seed with a minimal technique
	f.Add([]byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "attack-pattern",
				"name": "Test Technique",
				"description": "Test description",
				"x_mitre_domains": ["enterprise-attack"],
				"x_mitre_platforms": ["Linux"],
				"x_mitre_is_subtechnique": false,
				"external_references": [
					{
						"source_name": "mitre-attack",
						"external_id": "T1000"
					}
				],
				"kill_chain_phases": [
					{
						"kill_chain_name": "mitre-attack",
						"phase_name": "test"
					}
				]
			}
		]
	}`))

	// Seed with a minimal sub-technique
	f.Add([]byte(`{
		"type": "bundle",
		"objects": [
			{
				"type": "attack-pattern",
				"name": "Parent Technique",
				"description": "Parent description",
				"x_mitre_domains": ["enterprise-attack"],
				"x_mitre_platforms": ["Linux"],
				"x_mitre_is_subtechnique": false,
				"external_references": [
					{
						"source_name": "mitre-attack",
						"external_id": "T1000"
					}
				]
			},
			{
				"type": "attack-pattern",
				"name": "Sub Technique",
				"description": "Sub description",
				"x_mitre_domains": ["enterprise-attack"],
				"x_mitre_platforms": ["Linux"],
				"x_mitre_is_subtechnique": true,
				"external_references": [
					{
						"source_name": "mitre-attack",
						"external_id": "T1000.001"
					}
				]
			}
		]
	}`))

	// Seed with various invalid/edge case JSON
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"objects":null}`))
	f.Add([]byte(`{"objects":[{}]}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Add([]byte(`{malformed`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test should never panic, regardless of input
		// We don't care about the error, just that it doesn't panic
		_, _ = UnmarshalAndExtractMitreAttackBundle(Enterprise, []Platform{Linux}, data)
	})
}

// FuzzExtractMitreAttackBundle tests that ExtractMitreAttackBundle
// never panics on arbitrary mitreObject arrays and edge cases.
func FuzzExtractMitreAttackBundle(f *testing.F) {
	// Since we can't directly fuzz complex types, we'll fuzz JSON and unmarshal it
	// This tests both the unmarshaling and extraction logic paths

	// Empty objects array
	f.Add([]byte(`{"objects":[]}`))

	// Single object with missing fields
	f.Add([]byte(`{"objects":[{"type":"x-mitre-tactic"}]}`))

	// Multiple objects with different types
	f.Add([]byte(`{"objects":[
		{"type":"x-mitre-collection","x_mitre_version":"1.0"},
		{"type":"x-mitre-tactic","name":"Tactic1"},
		{"type":"attack-pattern","name":"Tech1"}
	]}`))

	// Objects with empty/null arrays
	f.Add([]byte(`{"objects":[{
		"type":"attack-pattern",
		"x_mitre_domains":[],
		"x_mitre_platforms":[],
		"external_references":[],
		"kill_chain_phases":[]
	}]}`))

	// Sub-technique without parent
	f.Add([]byte(`{"objects":[{
		"type":"attack-pattern",
		"x_mitre_is_subtechnique":true,
		"x_mitre_domains":["enterprise-attack"],
		"x_mitre_platforms":["Linux"],
		"external_references":[{"source_name":"mitre-attack","external_id":"T9999.001"}]
	}]}`))

	// Technique with non-mitre-attack kill chain
	f.Add([]byte(`{"objects":[{
		"type":"attack-pattern",
		"x_mitre_domains":["enterprise-attack"],
		"x_mitre_platforms":["Linux"],
		"external_references":[{"source_name":"mitre-attack","external_id":"T1234"}],
		"kill_chain_phases":[{"kill_chain_name":"other-chain","phase_name":"test"}]
	}]}`))

	// Technique referencing non-existent tactic
	f.Add([]byte(`{"objects":[{
		"type":"attack-pattern",
		"x_mitre_domains":["enterprise-attack"],
		"x_mitre_platforms":["Linux"],
		"external_references":[{"source_name":"mitre-attack","external_id":"T1234"}],
		"kill_chain_phases":[{"kill_chain_name":"mitre-attack","phase_name":"nonexistent"}]
	}]}`))

	// Tactic with missing external_id
	f.Add([]byte(`{"objects":[{
		"type":"x-mitre-tactic",
		"name":"Test",
		"x_mitre_domains":["enterprise-attack"],
		"x_mitre_shortname":"test",
		"external_references":[{"source_name":"other-source"}]
	}]}`))

	// Unicode and special characters
	f.Add([]byte(`{"objects":[{
		"type":"attack-pattern",
		"name":"Test 测试 🔒",
		"description":"Description with\nnewlines\tand\ttabs",
		"x_mitre_domains":["enterprise-attack"],
		"x_mitre_platforms":["Linux"]
	}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Unmarshal the fuzzed JSON into mitreBundle
		var bundle mitreBundle
		if err := json.Unmarshal(data, &bundle); err != nil {
			// If unmarshaling fails, that's fine - we're testing robustness
			return
		}

		// Test extraction with various domain/platform combinations
		// Should never panic
		_, _ = ExtractMitreAttackBundle(Enterprise, []Platform{Linux}, bundle.Objects)
		_, _ = ExtractMitreAttackBundle(Mobile, []Platform{Android}, bundle.Objects)
		_, _ = ExtractMitreAttackBundle(Enterprise, []Platform{}, bundle.Objects)
		_, _ = ExtractMitreAttackBundle(Enterprise, []Platform{Linux, Windows, Container}, bundle.Objects)
	})
}

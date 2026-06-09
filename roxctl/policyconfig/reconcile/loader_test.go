package reconcile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePolicyDocuments_SinglePolicy(t *testing.T) {
	data := []byte(`
policyName: "Test Policy"
description: "A test policy"
categories:
  - "Security Best Practices"
lifecycleStages:
  - DEPLOY
severity: HIGH_SEVERITY
policySections:
  - sectionName: "section1"
    policyGroups:
      - fieldName: "Process UID"
        values:
          - value: "0"
`)
	specs, err := parsePolicyDocuments(data)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "Test Policy", specs[0].PolicyName)
	assert.Equal(t, "A test policy", specs[0].Description)
	assert.Equal(t, "HIGH_SEVERITY", specs[0].Severity)
	assert.Len(t, specs[0].PolicySections, 1)
}

func TestParsePolicyDocuments_MultiDocument(t *testing.T) {
	data := []byte(`
policyName: "Policy One"
categories:
  - "cat1"
lifecycleStages:
  - DEPLOY
severity: LOW_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f1"
        values:
          - value: "v1"
---
policyName: "Policy Two"
categories:
  - "cat2"
lifecycleStages:
  - BUILD
severity: MEDIUM_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f2"
        values:
          - value: "v2"
`)
	specs, err := parsePolicyDocuments(data)
	require.NoError(t, err)
	require.Len(t, specs, 2)
	assert.Equal(t, "Policy One", specs[0].PolicyName)
	assert.Equal(t, "Policy Two", specs[1].PolicyName)
}

func TestParsePolicyDocuments_SkipsEmptyDocuments(t *testing.T) {
	data := []byte(`
---
policyName: "Only Policy"
categories:
  - "cat1"
lifecycleStages:
  - DEPLOY
severity: LOW_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f1"
        values:
          - value: "v1"
---
`)
	specs, err := parsePolicyDocuments(data)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	assert.Equal(t, "Only Policy", specs[0].PolicyName)
}

func TestParsePolicyDocuments_SkipsDocumentsWithoutPolicyName(t *testing.T) {
	data := []byte(`
description: "has no policyName"
categories:
  - "cat1"
`)
	specs, err := parsePolicyDocuments(data)
	require.NoError(t, err)
	assert.Empty(t, specs)
}

func TestParsePolicyDocuments_InvalidYAML(t *testing.T) {
	data := []byte(`
policyName: "bad
  yaml: [unterminated
`)
	_, err := parsePolicyDocuments(data)
	assert.Error(t, err)
}

func TestLoadPoliciesFromDir(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "policy1.yaml"), `
policyName: "Policy A"
categories: ["cat1"]
lifecycleStages: ["DEPLOY"]
severity: HIGH_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f1"
        values: [{value: "v1"}]
`)

	writeFile(t, filepath.Join(dir, "policy2.yml"), `
policyName: "Policy B"
categories: ["cat1"]
lifecycleStages: ["BUILD"]
severity: LOW_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f2"
        values: [{value: "v2"}]
`)

	writeFile(t, filepath.Join(dir, "readme.txt"), "not a yaml file")

	specs, err := loadPoliciesFromDir(dir)
	require.NoError(t, err)
	require.Len(t, specs, 2)

	names := map[string]bool{}
	for _, s := range specs {
		names[s.PolicyName] = true
	}
	assert.True(t, names["Policy A"])
	assert.True(t, names["Policy B"])
}

func TestLoadPoliciesFromDir_DuplicateName(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.yaml"), `
policyName: "Same Name"
categories: ["cat1"]
lifecycleStages: ["DEPLOY"]
severity: HIGH_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f1"
        values: [{value: "v1"}]
`)

	writeFile(t, filepath.Join(dir, "b.yaml"), `
policyName: "Same Name"
categories: ["cat2"]
lifecycleStages: ["BUILD"]
severity: LOW_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f2"
        values: [{value: "v2"}]
`)

	_, err := loadPoliciesFromDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate policy name")
}

func TestLoadPoliciesFromDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	specs, err := loadPoliciesFromDir(dir)
	require.NoError(t, err)
	assert.Empty(t, specs)
}

func TestLoadPoliciesFromDir_Recursive(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	writeFile(t, filepath.Join(dir, "top.yaml"), `
policyName: "Top Level"
categories: ["cat1"]
lifecycleStages: ["DEPLOY"]
severity: HIGH_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f1"
        values: [{value: "v1"}]
`)

	writeFile(t, filepath.Join(subdir, "nested.yaml"), `
policyName: "Nested"
categories: ["cat1"]
lifecycleStages: ["DEPLOY"]
severity: LOW_SEVERITY
policySections:
  - policyGroups:
      - fieldName: "f2"
        values: [{value: "v2"}]
`)

	specs, err := loadPoliciesFromDir(dir)
	require.NoError(t, err)
	require.Len(t, specs, 2)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

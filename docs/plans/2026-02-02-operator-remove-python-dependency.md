# Operator: Remove Python Dependency Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Python-based CSV patching tools (`patch-csv.py`, `fix-spec-descriptor-order.py`) with Go implementations to eliminate Python dependency from operator build.

**Architecture:** Create two standalone Go CLI tools in `operator/cmd/` that read CSV YAML from stdin and write to stdout. Use `sigs.k8s.io/yaml` (already in project deps) for YAML handling. Port Python unit tests to Go table-driven tests. Validate outputs match using `operator-sdk bundle validate`. Update Makefile to build and use Go tools, update Konflux Dockerfile to remove Python base image.

**Tech Stack:** Go 1.25, sigs.k8s.io/yaml v1.6.0, testify v1.11.1, operator-sdk for validation

---

## Task 1: Create csv-patcher CLI - Version Types

**Files:**
- Create: `operator/cmd/csv-patcher/main.go`
- Create: `operator/cmd/csv-patcher/version.go`
- Create: `operator/cmd/csv-patcher/version_test.go`

**Step 1: Write the failing test for XyzVersion parsing**

Create `operator/cmd/csv-patcher/version_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXyzVersion_ParseFrom(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    XyzVersion
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "3.74.0",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:  "with patch",
			input: "4.1.2",
			want:  XyzVersion{X: 4, Y: 1, Z: 2},
		},
		{
			name:  "with build suffix",
			input: "3.74.0-123",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:  "nightly build",
			input: "3.74.x-nightly-20230224",
			want:  XyzVersion{X: 3, Y: 74, Z: 0},
		},
		{
			name:    "invalid format",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseXyzVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestXyzVersion_String(t *testing.T) {
	v := XyzVersion{X: 3, Y: 74, Z: 2}
	assert.Equal(t, "3.74.2", v.String())
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/csv-patcher
go test -v
```

Expected: FAIL - undefined: XyzVersion, ParseXyzVersion

**Step 3: Write minimal implementation**

Create `operator/cmd/csv-patcher/version.go`:

```go
package main

import (
	"fmt"
	"regexp"
	"strconv"
)

// XyzVersion represents a semantic version with major.minor.patch components
type XyzVersion struct {
	X int // Major version
	Y int // Minor version
	Z int // Patch version
}

var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(x|\d+)(-.+)?$`)

// ParseXyzVersion parses a version string into XyzVersion
// Supports formats: "3.74.0", "3.74.0-123", "3.74.x-nightly-20230224"
func ParseXyzVersion(versionStr string) (XyzVersion, error) {
	matches := versionRegex.FindStringSubmatch(versionStr)
	if matches == nil {
		return XyzVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	x, _ := strconv.Atoi(matches[1])
	y, _ := strconv.Atoi(matches[2])

	z := 0
	if matches[3] != "x" {
		z, _ = strconv.Atoi(matches[3])
	}

	return XyzVersion{X: x, Y: y, Z: z}, nil
}

// String returns the version as "x.y.z"
func (v XyzVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.X, v.Y, v.Z)
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other
func (v XyzVersion) Compare(other XyzVersion) int {
	if v.X != other.X {
		if v.X < other.X {
			return -1
		}
		return 1
	}
	if v.Y != other.Y {
		if v.Y < other.Y {
			return -1
		}
		return 1
	}
	if v.Z != other.Z {
		if v.Z < other.Z {
			return -1
		}
		return 1
	}
	return 0
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/csv-patcher
go test -v
```

Expected: PASS - all version parsing tests pass

**Step 5: Commit**

```bash
git add operator/cmd/csv-patcher/version.go operator/cmd/csv-patcher/version_test.go
git commit -m "feat(operator): add XyzVersion type for CSV version handling

- Add XyzVersion struct with X.Y.Z components
- Support parsing version strings with build suffixes
- Support 'x' as patch placeholder (converted to 0)
- Add comparison method for version ordering"
```

---

## Task 2: Create csv-patcher CLI - Previous Y-Stream Calculation

**Files:**
- Modify: `operator/cmd/csv-patcher/version.go`
- Modify: `operator/cmd/csv-patcher/version_test.go`

**Step 1: Write the failing test for GetPreviousYStream**

Add to `operator/cmd/csv-patcher/version_test.go`:

```go
func TestGetPreviousYStream(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "minor version decrement",
			version: "3.74.0",
			want:    "3.73.0",
		},
		{
			name:    "minor version decrement with patch",
			version: "3.74.3",
			want:    "3.73.0",
		},
		{
			name:    "major version 4 to 3.74.0",
			version: "4.0.0",
			want:    "3.74.0",
		},
		{
			name:    "major version 4 minor 1",
			version: "4.1.0",
			want:    "4.0.0",
		},
		{
			name:    "trunk builds",
			version: "1.0.0",
			want:    "0.0.0",
		},
		{
			name:    "with nightly suffix",
			version: "3.74.x-nightly-20230224",
			want:    "3.73.0",
		},
		{
			name:    "unknown major version",
			version: "99.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPreviousYStream(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestGetPreviousYStream
```

Expected: FAIL - undefined: GetPreviousYStream

**Step 3: Write minimal implementation**

Add to `operator/cmd/csv-patcher/version.go`:

```go
// GetPreviousYStream returns the previous Y-Stream version
// Y-Stream versions have patch number = 0 (e.g., 3.73.0, 3.74.0, 4.0.0)
// This implements the logic from scripts/get-previous-y-stream.sh
func GetPreviousYStream(versionStr string) (string, error) {
	v, err := ParseXyzVersion(versionStr)
	if err != nil {
		return "", err
	}

	if v.Y > 0 {
		// If minor version > 0, previous Y-Stream is one minor less
		return fmt.Sprintf("%d.%d.0", v.X, v.Y-1), nil
	}

	// For major version bumps, maintain hardcoded mapping
	switch v.X {
	case 4:
		return "3.74.0", nil
	case 1:
		// 0.0.0 was never released, but used for trunk builds
		return "0.0.0", nil
	default:
		return "", fmt.Errorf("don't know the previous Y-Stream for %d.%d", v.X, v.Y)
	}
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestGetPreviousYStream
```

Expected: PASS - all Y-Stream calculation tests pass

**Step 5: Commit**

```bash
git add operator/cmd/csv-patcher/version.go operator/cmd/csv-patcher/version_test.go
git commit -m "feat(operator): add Y-Stream version calculation

- Implement GetPreviousYStream matching get-previous-y-stream.sh logic
- Handle minor version decrements (3.74.0 -> 3.73.0)
- Handle major version transitions (4.0.0 -> 3.74.0)
- Handle trunk builds (1.0.0 -> 0.0.0)"
```

---

## Task 3: Create csv-patcher CLI - Replace Version Calculation

**Files:**
- Modify: `operator/cmd/csv-patcher/version.go`
- Modify: `operator/cmd/csv-patcher/version_test.go`

**Step 1: Write the failing test for CalculateReplacedVersion**

Add to `operator/cmd/csv-patcher/version_test.go`:

```go
func TestCalculateReplacedVersion(t *testing.T) {
	tests := []struct {
		name          string
		current       string
		first         string
		previous      string
		skips         []string
		unreleased    string
		want          string
		wantNil       bool
	}{
		{
			name:     "downstream trunk builds get no replace",
			current:  "1.0.0",
			first:    "4.0.0",
			previous: "0.0.0",
			wantNil:  true,
		},
		{
			name:     "first release gets no replace",
			current:  "4.0.0",
			first:    "4.0.0",
			previous: "3.74.0",
			wantNil:  true,
		},
		{
			name:     "patch follows normal release",
			current:  "4.0.1",
			first:    "4.0.0",
			previous: "3.74.0",
			want:     "4.0.0",
		},
		{
			name:     "Y-Stream release replaces previous Y-Stream",
			current:  "4.2.0",
			first:    "4.0.0",
			previous: "4.1.0",
			want:     "4.1.0",
		},
		{
			name:     "normal patch replaces previous patch",
			current:  "4.1.3",
			first:    "4.0.0",
			previous: "4.0.0",
			want:     "4.1.2",
		},
		{
			name:     "first patch replaces its Y-Stream",
			current:  "4.1.1",
			first:    "4.0.0",
			previous: "4.0.0",
			want:     "4.1.0",
		},
		{
			name:     "skipped immediate preceding patch still used",
			current:  "4.1.1",
			first:    "4.0.0",
			previous: "4.0.0",
			skips:    []string{"4.1.0"},
			want:     "4.1.0",
		},
		{
			name:     "skipped immediate preceding minor patch still used",
			current:  "4.1.3",
			first:    "4.0.0",
			previous: "4.0.0",
			skips:    []string{"4.1.2"},
			want:     "4.1.2",
		},
		{
			name:     "skipped previous Y-Stream targets next patch",
			current:  "4.2.0",
			first:    "4.0.0",
			previous: "4.1.0",
			skips:    []string{"4.1.0"},
			want:     "4.1.1",
		},
		{
			name:     "multiple skips iterate to find non-skipped",
			current:  "4.3.0",
			first:    "4.0.0",
			previous: "4.2.0",
			skips:    []string{"4.1.0", "4.2.0", "4.2.1", "4.2.2", "4.4.0"},
			want:     "4.2.3",
		},
		{
			name:       "unreleased version fallback",
			current:    "4.2.0",
			first:      "4.0.0",
			previous:   "4.1.0",
			unreleased: "4.1.0",
			want:       "4.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skipVersions := make([]XyzVersion, 0)
			for _, s := range tt.skips {
				v, err := ParseXyzVersion(s)
				require.NoError(t, err)
				skipVersions = append(skipVersions, v)
			}

			got, err := CalculateReplacedVersion(tt.current, tt.first, tt.previous, skipVersions, tt.unreleased)
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want, got.String())
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestCalculateReplacedVersion
```

Expected: FAIL - undefined: CalculateReplacedVersion

**Step 3: Write minimal implementation**

Add to `operator/cmd/csv-patcher/version.go`:

```go
// CalculateReplacedVersion determines which version this release replaces
// This is complex logic that handles Y-Stream vs patch releases, version skips, and unreleased versions
func CalculateReplacedVersion(current, first, previousYStream string, skips []XyzVersion, unreleased string) (*XyzVersion, error) {
	currentXyz, err := ParseXyzVersion(current)
	if err != nil {
		return nil, err
	}

	firstXyz, err := ParseXyzVersion(first)
	if err != nil {
		return nil, err
	}

	previousXyz, err := ParseXyzVersion(previousYStream)
	if err != nil {
		return nil, err
	}

	// First version or earlier gets no replace
	if currentXyz.Compare(firstXyz) <= 0 {
		return nil, nil
	}

	// Determine initial replace candidate
	var initialReplace XyzVersion
	if currentXyz.Z == 0 {
		// New minor release replaces previous minor (e.g., 4.2.0 replaces 4.1.0)
		initialReplace = previousXyz
	} else {
		// Patch replaces previous patch (e.g., 4.2.2 replaces 4.2.1)
		initialReplace = XyzVersion{X: currentXyz.X, Y: currentXyz.Y, Z: currentXyz.Z - 1}
	}

	// If initial replace is unreleased, try previous one
	if unreleased != "" && initialReplace.String() == unreleased {
		prev, err := GetPreviousYStream(initialReplace.String())
		if err != nil {
			return nil, err
		}
		initialReplace, err = ParseXyzVersion(prev)
		if err != nil {
			return nil, err
		}
	}

	currentReplace := initialReplace

	// Skip over broken versions in the skips list
	skipMap := make(map[string]bool)
	for _, skip := range skips {
		skipMap[skip.String()] = true
	}

	for skipMap[currentReplace.String()] {
		// Try next patch
		currentReplace = XyzVersion{X: currentReplace.X, Y: currentReplace.Y, Z: currentReplace.Z + 1}
	}

	// Exception: if we're releasing immediate patch to broken version, still replace it
	// E.g., 4.1.0 is broken and in skips, 4.1.1 still replaces 4.1.0
	// This works because 4.1.1 will have skipRange allowing upgrade from 4.0.0
	if currentReplace.Compare(currentXyz) >= 0 {
		currentReplace = initialReplace
	}

	return &currentReplace, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestCalculateReplacedVersion
```

Expected: PASS - all replace version calculation tests pass

**Step 5: Commit**

```bash
git add operator/cmd/csv-patcher/version.go operator/cmd/csv-patcher/version_test.go
git commit -m "feat(operator): add replaced version calculation logic

- Implement CalculateReplacedVersion matching Python logic
- Handle Y-Stream vs patch release logic
- Skip broken versions from skips list
- Handle unreleased version fallback
- Exception for immediate patch to broken version"
```

---

## Task 4: Create csv-patcher CLI - CSV Structure Types

**Files:**
- Create: `operator/cmd/csv-patcher/csv.go`

**Step 1: Write CSV structure types**

Create `operator/cmd/csv-patcher/csv.go`:

```go
package main

// csvDocument represents the ClusterServiceVersion YAML structure
// We only define fields we need to modify, using map[string]interface{} for the rest
type csvDocument struct {
	Metadata struct {
		Name        string                 `yaml:"name"`
		Annotations map[string]interface{} `yaml:"annotations"`
		Labels      map[string]interface{} `yaml:"labels,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Version       string                   `yaml:"version"`
		Replaces      string                   `yaml:"replaces,omitempty"`
		Skips         []string                 `yaml:"skips,omitempty"`
		RelatedImages []map[string]interface{} `yaml:"relatedImages,omitempty"`
		CustomResourceDefinitions struct {
			Owned []map[string]interface{} `yaml:"owned"`
		} `yaml:"customresourcedefinitions"`
		// Keep other fields as-is
		Rest map[string]interface{} `yaml:",inline"`
	} `yaml:"spec"`
}

// relatedImage represents an entry in spec.relatedImages
type relatedImage struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}
```

**Step 2: No test needed (pure data structures)**

This step defines types only, no behavior to test yet.

**Step 3: Commit**

```bash
git add operator/cmd/csv-patcher/csv.go
git commit -m "feat(operator): add CSV document structure types

- Define csvDocument struct matching CSV YAML schema
- Use map[string]interface{} for fields we don't modify
- Add relatedImage type for spec.relatedImages entries"
```

---

## Task 5: Create csv-patcher CLI - String Replacement Utility

**Files:**
- Create: `operator/cmd/csv-patcher/rewrite.go`
- Create: `operator/cmd/csv-patcher/rewrite_test.go`

**Step 1: Write the failing test for rewriteStrings**

Create `operator/cmd/csv-patcher/rewrite_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		old      string
		new      string
		expected interface{}
		modified bool
	}{
		{
			name:     "replace string value",
			input:    "quay.io/stackrox-io/stackrox-operator:0.0.1",
			old:      "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new:      "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			modified: true,
		},
		{
			name:     "no match leaves unchanged",
			input:    "some-other-value",
			old:      "not-found",
			new:      "replacement",
			expected: "some-other-value",
			modified: false,
		},
		{
			name: "replace in map values",
			input: map[string]interface{}{
				"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
				"other":          "unchanged",
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: map[string]interface{}{
				"containerImage": "quay.io/stackrox-io/stackrox-operator:4.0.0",
				"other":          "unchanged",
			},
			modified: true,
		},
		{
			name: "replace in slice elements",
			input: []interface{}{
				"quay.io/stackrox-io/stackrox-operator:0.0.1",
				"other-value",
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: []interface{}{
				"quay.io/stackrox-io/stackrox-operator:4.0.0",
				"other-value",
			},
			modified: true,
		},
		{
			name: "replace in nested structures",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": []interface{}{
						"quay.io/stackrox-io/stackrox-operator:0.0.1",
					},
				},
			},
			old: "quay.io/stackrox-io/stackrox-operator:0.0.1",
			new: "quay.io/stackrox-io/stackrox-operator:4.0.0",
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": []interface{}{
						"quay.io/stackrox-io/stackrox-operator:4.0.0",
					},
				},
			},
			modified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := rewriteStrings(tt.input, tt.old, tt.new)
			assert.Equal(t, tt.modified, modified)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestRewriteStrings
```

Expected: FAIL - undefined: rewriteStrings

**Step 3: Write minimal implementation**

Create `operator/cmd/csv-patcher/rewrite.go`:

```go
package main

// rewriteStrings recursively traverses data structures and replaces all
// string values matching 'old' with 'new'
// Returns true if any replacements were made
func rewriteStrings(data interface{}, old, new string) bool {
	modified := false

	switch v := data.(type) {
	case string:
		// Can't modify strings in place, caller must handle
		return false

	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[key] = new
				modified = true
			} else if rewriteStrings(value, old, new) {
				modified = true
			}
		}

	case []interface{}:
		for i, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[i] = new
				modified = true
			} else if rewriteStrings(value, old, new) {
				modified = true
			}
		}
	}

	return modified
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestRewriteStrings
```

Expected: PASS - all rewrite tests pass

**Step 5: Commit**

```bash
git add operator/cmd/csv-patcher/rewrite.go operator/cmd/csv-patcher/rewrite_test.go
git commit -m "feat(operator): add recursive string replacement utility

- Implement rewriteStrings for deep structure traversal
- Replace matching strings in maps, slices, nested structures
- Return modification flag for logging"
```

---

## Task 6: Create csv-patcher CLI - Main Patching Logic

**Files:**
- Create: `operator/cmd/csv-patcher/patch.go`
- Create: `operator/cmd/csv-patcher/patch_test.go`

**Step 1: Write the failing test for PatchCSV**

Create `operator/cmd/csv-patcher/patch_test.go`:

```go
package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchCSV(t *testing.T) {
	// Set up test environment variables
	os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.0.0")
	os.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.0.0")
	defer func() {
		os.Unsetenv("RELATED_IMAGE_MAIN")
		os.Unsetenv("RELATED_IMAGE_SCANNER")
	}()

	tests := []struct {
		name       string
		input      map[string]interface{}
		opts       PatchOptions
		wantErr    bool
		assertions func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "basic version patching",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "rhacs-operator.v0.0.1",
					"annotations": map[string]interface{}{
						"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
						"createdAt":      "",
					},
				},
				"spec": map[string]interface{}{
					"version": "0.0.1",
				},
			},
			opts: PatchOptions{
				Version:        "4.0.0",
				OperatorImage:  "quay.io/stackrox-io/stackrox-operator:4.0.0",
				FirstVersion:   "3.62.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result map[string]interface{}) {
				metadata := result["metadata"].(map[string]interface{})
				assert.Equal(t, "rhacs-operator.v4.0.0", metadata["name"])

				annotations := metadata["annotations"].(map[string]interface{})
				assert.Equal(t, "quay.io/stackrox-io/stackrox-operator:4.0.0", annotations["containerImage"])
				assert.NotEmpty(t, annotations["createdAt"])

				spec := result["spec"].(map[string]interface{})
				assert.Equal(t, "4.0.0", spec["version"])
			},
		},
		{
			name: "replaces version calculation",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "rhacs-operator.v0.0.1",
					"annotations": map[string]interface{}{
						"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
					},
				},
				"spec": map[string]interface{}{
					"version": "0.0.1",
				},
			},
			opts: PatchOptions{
				Version:        "4.0.1",
				OperatorImage:  "quay.io/stackrox-io/stackrox-operator:4.0.1",
				FirstVersion:   "4.0.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result map[string]interface{}) {
				spec := result["spec"].(map[string]interface{})
				assert.Equal(t, "rhacs-operator.v4.0.0", spec["replaces"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PatchCSV(tt.input, tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.assertions != nil {
				tt.assertions(t, tt.input)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestPatchCSV
```

Expected: FAIL - undefined: PatchCSV, PatchOptions

**Step 3: Write minimal implementation**

Create `operator/cmd/csv-patcher/patch.go`:

```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// PatchOptions contains all options for patching a CSV
type PatchOptions struct {
	Version            string
	OperatorImage      string
	FirstVersion       string
	RelatedImagesMode  string
	ExtraSupportedArchs []string
	Unreleased         string
}

// PatchCSV modifies the CSV document in-place according to options
func PatchCSV(doc map[string]interface{}, opts PatchOptions) error {
	// Update createdAt timestamp
	metadata := doc["metadata"].(map[string]interface{})
	annotations := metadata["annotations"].(map[string]interface{})
	annotations["createdAt"] = time.Now().UTC().Format(time.RFC3339)

	// Replace placeholder image with actual operator image
	placeholderImage := annotations["containerImage"].(string)
	rewriteStrings(doc, placeholderImage, opts.OperatorImage)

	// Update metadata name with version
	rawName := strings.TrimSuffix(metadata["name"].(string), ".v0.0.1")
	if !strings.HasSuffix(metadata["name"].(string), ".v0.0.1") {
		return fmt.Errorf("metadata.name does not end with .v0.0.1: %s", metadata["name"])
	}
	metadata["name"] = fmt.Sprintf("%s.v%s", rawName, opts.Version)

	// Update spec.version
	spec := doc["spec"].(map[string]interface{})
	spec["version"] = opts.Version

	// Handle related images based on mode
	switch opts.RelatedImagesMode {
	case "downstream":
		if err := injectRelatedImageEnvVars(spec); err != nil {
			return err
		}
		delete(spec, "relatedImages")
	case "omit":
		delete(spec, "relatedImages")
	case "konflux":
		if err := constructRelatedImages(spec, opts.OperatorImage); err != nil {
			return err
		}
	}

	// Calculate previous Y-Stream
	previousYStream, err := GetPreviousYStream(opts.Version)
	if err != nil {
		return err
	}

	// Set olm.skipRange
	annotations["olm.skipRange"] = fmt.Sprintf(">= %s < %s", previousYStream, opts.Version)

	// Add multi-arch labels
	if metadata["labels"] == nil {
		metadata["labels"] = make(map[string]interface{})
	}
	labels := metadata["labels"].(map[string]interface{})
	for _, arch := range opts.ExtraSupportedArchs {
		labels[fmt.Sprintf("operatorframework.io/arch.%s", arch)] = "supported"
	}

	// Parse skips
	skips := make([]XyzVersion, 0)
	if rawSkips, ok := spec["skips"].([]interface{}); ok {
		for _, s := range rawSkips {
			skipStr := s.(string)
			skipVer := strings.TrimPrefix(skipStr, rawName+".v")
			v, err := ParseXyzVersion(skipVer)
			if err != nil {
				return err
			}
			skips = append(skips, v)
		}
	}

	// Calculate replaced version
	replacedVersion, err := CalculateReplacedVersion(
		opts.Version,
		opts.FirstVersion,
		previousYStream,
		skips,
		opts.Unreleased,
	)
	if err != nil {
		return err
	}

	if replacedVersion != nil {
		spec["replaces"] = fmt.Sprintf("%s.v%s", rawName, replacedVersion.String())
	}

	// Add SecurityPolicy CRD
	if err := addSecurityPolicyCRD(spec); err != nil {
		return err
	}

	return nil
}

func injectRelatedImageEnvVars(spec map[string]interface{}) error {
	// Find all RELATED_IMAGE_* env vars in the spec and replace with actual values
	var traverse func(interface{}) error
	traverse = func(data interface{}) error {
		switch v := data.(type) {
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok && strings.HasPrefix(name, "RELATED_IMAGE_") {
				envValue := os.Getenv(name)
				if envValue == "" {
					return fmt.Errorf("required environment variable %s is not set", name)
				}
				v["value"] = envValue
			}
			for _, value := range v {
				if err := traverse(value); err != nil {
					return err
				}
			}
		case []interface{}:
			for _, value := range v {
				if err := traverse(value); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return traverse(spec)
}

func constructRelatedImages(spec map[string]interface{}, managerImage string) error {
	relatedImages := make([]map[string]interface{}, 0)

	// Collect all RELATED_IMAGE_* env vars
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "RELATED_IMAGE_") {
			parts := strings.SplitN(envVar, "=", 2)
			name := strings.TrimPrefix(parts[0], "RELATED_IMAGE_")
			name = strings.ToLower(name)
			image := parts[1]

			relatedImages = append(relatedImages, map[string]interface{}{
				"name":  name,
				"image": image,
			})
		}
	}

	// Add manager image
	relatedImages = append(relatedImages, map[string]interface{}{
		"name":  "manager",
		"image": managerImage,
	})

	spec["relatedImages"] = relatedImages
	return nil
}

func addSecurityPolicyCRD(spec map[string]interface{}) error {
	crd := map[string]interface{}{
		"name":        "securitypolicies.config.stackrox.io",
		"version":     "v1alpha1",
		"kind":        "SecurityPolicy",
		"displayName": "Security Policy",
		"description": "SecurityPolicy is the schema for the policies API.",
		"resources": []map[string]interface{}{
			{
				"kind":    "Deployment",
				"name":    "",
				"version": "v1",
			},
		},
	}

	crds := spec["customresourcedefinitions"].(map[string]interface{})
	owned := crds["owned"].([]interface{})
	crds["owned"] = append(owned, crd)

	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/csv-patcher
go test -v -run TestPatchCSV
```

Expected: PASS - all CSV patching tests pass

**Step 5: Commit**

```bash
git add operator/cmd/csv-patcher/patch.go operator/cmd/csv-patcher/patch_test.go
git commit -m "feat(operator): add CSV patching logic

- Implement PatchCSV with all transformation logic
- Update version, name, timestamps, images
- Calculate replaces version with skip handling
- Handle related images in 3 modes (downstream/omit/konflux)
- Add multi-arch labels
- Add SecurityPolicy CRD to owned CRDs"
```

---

## Task 7: Create csv-patcher CLI - Main Entry Point

**Files:**
- Modify: `operator/cmd/csv-patcher/main.go`

**Step 1: Write main function and CLI parsing**

Update `operator/cmd/csv-patcher/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

func main() {
	// Parse command-line flags
	version := flag.String("use-version", "", "SemVer version of the operator (required)")
	firstVersion := flag.String("first-version", "", "First version of operator ever published (required)")
	operatorImage := flag.String("operator-image", "", "Operator image reference (required)")
	relatedImagesMode := flag.String("related-images-mode", "downstream", "Mode for related images: downstream, omit, konflux")
	addSupportedArch := flag.String("add-supported-arch", "amd64,arm64,ppc64le,s390x", "Comma-separated list of supported architectures")
	echoReplacedVersionOnly := flag.Bool("echo-replaced-version-only", false, "Only compute and print replaced version")
	unreleased := flag.String("unreleased", "", "Not yet released version, if any")

	flag.Parse()

	if *version == "" || *firstVersion == "" || *operatorImage == "" {
		fmt.Fprintln(os.Stderr, "Error: --use-version, --first-version, and --operator-image are required")
		flag.Usage()
		os.Exit(1)
	}

	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal(input, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Handle --echo-replaced-version-only mode
	if *echoReplacedVersionOnly {
		if err := echoReplacedVersion(doc, *version, *firstVersion, *unreleased); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Parse supported architectures
	var arches []string
	if *addSupportedArch != "" {
		arches = splitComma(*addSupportedArch)
	}

	// Patch the CSV
	opts := PatchOptions{
		Version:             *version,
		OperatorImage:       *operatorImage,
		FirstVersion:        *firstVersion,
		RelatedImagesMode:   *relatedImagesMode,
		ExtraSupportedArchs: arches,
		Unreleased:          *unreleased,
	}

	if err := PatchCSV(doc, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error patching CSV: %v\n", err)
		os.Exit(1)
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
		os.Exit(1)
	}

	// Write to stdout
	fmt.Print(string(output))
}

func echoReplacedVersion(doc map[string]interface{}, version, firstVersion, unreleased string) error {
	metadata := doc["metadata"].(map[string]interface{})
	name := metadata["name"].(string)

	rawName := ""
	if name == "rhacs-operator.v0.0.1" {
		rawName = "rhacs-operator"
	} else {
		return fmt.Errorf("unexpected metadata.name format: %s", name)
	}

	spec := doc["spec"].(map[string]interface{})
	skips := make([]XyzVersion, 0)
	if rawSkips, ok := spec["skips"].([]interface{}); ok {
		for _, s := range rawSkips {
			skipStr := s.(string)
			skipVer := ""
			if skipStr == rawName+".v0.0.1" {
				continue
			}
			// Extract version from "rhacs-operator.vX.Y.Z"
			parts := splitString(skipStr, ".")
			if len(parts) >= 2 {
				skipVer = parts[len(parts)-3] + "." + parts[len(parts)-2] + "." + parts[len(parts)-1]
				skipVer = trimPrefix(skipVer, "v")
			}

			v, err := ParseXyzVersion(skipVer)
			if err != nil {
				return err
			}
			skips = append(skips, v)
		}
	}

	previousYStream, err := GetPreviousYStream(version)
	if err != nil {
		return err
	}

	replacedVersion, err := CalculateReplacedVersion(version, firstVersion, previousYStream, skips, unreleased)
	if err != nil {
		return err
	}

	if replacedVersion != nil {
		fmt.Println(replacedVersion.String())
	}

	return nil
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for _, p := range splitString(s, ",") {
		if trimmed := trimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := []string{}
	current := ""
	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
```

**Step 2: Build and test manually**

```bash
cd operator
go build -o bin/csv-patcher ./cmd/csv-patcher
echo '{"metadata":{"name":"rhacs-operator.v0.0.1","annotations":{"containerImage":"old"}},"spec":{"version":"0.0.1"}}' | \
  ./bin/csv-patcher --use-version=4.0.0 --first-version=3.62.0 --operator-image=new
```

Expected: Outputs YAML with version 4.0.0 and image "new"

**Step 3: Commit**

```bash
git add operator/cmd/csv-patcher/main.go
git commit -m "feat(operator): add csv-patcher CLI entry point

- Parse command-line flags for all options
- Read CSV from stdin, write to stdout
- Support --echo-replaced-version-only mode
- Handle comma-separated architecture list"
```

---

## Task 8: Create fix-spec-descriptors CLI

**Files:**
- Create: `operator/cmd/fix-spec-descriptors/main.go`
- Create: `operator/cmd/fix-spec-descriptors/main_test.go`

**Step 1: Write the failing test for fix-spec-descriptors**

Create `operator/cmd/fix-spec-descriptors/main_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixDescriptorOrder(t *testing.T) {
	descriptors := []map[string]interface{}{
		{"path": "central.db.enabled"},
		{"path": "central"},
		{"path": "central.db"},
		{"path": "scanner.enabled"},
		{"path": "scanner"},
	}

	fixDescriptorOrder(descriptors)

	// Verify order: parents before children
	paths := make([]string, len(descriptors))
	for i, d := range descriptors {
		paths[i] = d["path"].(string)
	}

	assert.Equal(t, []string{
		"central",
		"central.db",
		"central.db.enabled",
		"scanner",
		"scanner.enabled",
	}, paths)
}

func TestAllowRelativeFieldDependencies(t *testing.T) {
	descriptors := []map[string]interface{}{
		{
			"path": "central.db.passwordSecret",
			"x-descriptors": []interface{}{
				"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true",
			},
		},
		{
			"path": "central.db.enabled",
		},
	}

	allowRelativeFieldDependencies(descriptors)

	xDescs := descriptors[0]["x-descriptors"].([]interface{})
	assert.Equal(t,
		"urn:alm:descriptor:com.tectonic.ui:fieldDependency:central.db.enabled:true",
		xDescs[0])
}
```

**Step 2: Run test to verify it fails**

```bash
cd operator/cmd/fix-spec-descriptors
go test -v
```

Expected: FAIL - undefined functions

**Step 3: Write minimal implementation**

Create `operator/cmd/fix-spec-descriptors/main.go`:

```go
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

func main() {
	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal(input, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Fix descriptors in all owned CRDs
	spec := doc["spec"].(map[string]interface{})
	crds := spec["customresourcedefinitions"].(map[string]interface{})
	owned := crds["owned"].([]interface{})

	for _, crd := range owned {
		crdMap := crd.(map[string]interface{})
		if specDescriptors, ok := crdMap["specDescriptors"].([]interface{}); ok {
			// Convert to []map[string]interface{}
			descriptors := make([]map[string]interface{}, len(specDescriptors))
			for i, d := range specDescriptors {
				descriptors[i] = d.(map[string]interface{})
			}

			fixDescriptorOrder(descriptors)
			allowRelativeFieldDependencies(descriptors)

			// No need to reassign, we modified in place
		}
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
		os.Exit(1)
	}

	// Write to stdout
	fmt.Print(string(output))
}

// fixDescriptorOrder performs stable sort so parents appear before children
func fixDescriptorOrder(descriptors []map[string]interface{}) {
	sort.SliceStable(descriptors, func(i, j int) bool {
		pathI := "." + descriptors[i]["path"].(string)
		pathJ := "." + descriptors[j]["path"].(string)

		// Extract parent path (everything before last dot)
		parentI := pathI[:strings.LastIndex(pathI, ".")]
		parentJ := pathJ[:strings.LastIndex(pathJ, ".")]

		return parentI < parentJ
	})
}

// allowRelativeFieldDependencies converts relative field dependencies to absolute
func allowRelativeFieldDependencies(descriptors []map[string]interface{}) {
	for _, d := range descriptors {
		xDescriptors, ok := d["x-descriptors"].([]interface{})
		if !ok {
			continue
		}

		for i, xDesc := range xDescriptors {
			xDescStr, ok := xDesc.(string)
			if !ok {
				continue
			}

			// Check if it's a fieldDependency descriptor
			if !strings.Contains(xDescStr, "urn:alm:descriptor:com.tectonic.ui:fieldDependency:") {
				continue
			}

			// Split to extract field and value
			parts := strings.Split(xDescStr, ":")
			if len(parts) < 7 {
				continue
			}

			field := parts[5]
			value := parts[6]

			// If field starts with '.', it's relative
			if !strings.HasPrefix(field, ".") {
				continue
			}

			// Resolve relative to current path
			currentPath := "." + d["path"].(string)
			parentPath := currentPath[:strings.LastIndex(currentPath, ".")]
			absoluteField := strings.TrimPrefix(parentPath, ".") + field

			// Reconstruct descriptor with absolute path
			xDescriptors[i] = fmt.Sprintf("urn:alm:descriptor:com.tectonic.ui:fieldDependency:%s:%s",
				absoluteField, value)
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
cd operator/cmd/fix-spec-descriptors
go test -v
```

Expected: PASS - all descriptor fixing tests pass

**Step 5: Commit**

```bash
git add operator/cmd/fix-spec-descriptors/
git commit -m "feat(operator): add fix-spec-descriptors CLI tool

- Sort spec descriptors so parents come before children
- Resolve relative field dependencies to absolute paths
- Read from stdin, write to stdout"
```

---

## Task 9: Add Go Tool Build Targets to Makefile

**Files:**
- Modify: `operator/Makefile`

**Step 1: Add Go tool build targets**

Add after the existing tool definitions (around line 226):

```makefile
# CSV patching Go tools
CSV_PATCHER := $(LOCALBIN)/csv-patcher
FIX_SPEC_DESCRIPTORS := $(LOCALBIN)/fix-spec-descriptors

$(CSV_PATCHER): cmd/csv-patcher/*.go
	@echo "+ $(notdir $@)"
	$(SILENT)cd cmd/csv-patcher && go build -o $@ .

$(FIX_SPEC_DESCRIPTORS): cmd/fix-spec-descriptors/*.go
	@echo "+ $(notdir $@)"
	$(SILENT)cd cmd/fix-spec-descriptors && go build -o $@ .

.PHONY: csv-patcher
csv-patcher: $(CSV_PATCHER) ## Build csv-patcher tool

.PHONY: fix-spec-descriptors
fix-spec-descriptors: $(FIX_SPEC_DESCRIPTORS) ## Build fix-spec-descriptors tool
```

**Step 2: Test building the tools**

```bash
cd operator
make csv-patcher fix-spec-descriptors
```

Expected: Both binaries built successfully in `operator/bin/`

**Step 3: Commit**

```bash
git add operator/Makefile
git commit -m "feat(operator): add Makefile targets for Go CSV tools

- Add csv-patcher build target
- Add fix-spec-descriptors build target
- Tools built to LOCALBIN like other dev tools"
```

---

## Task 10: Add Feature Flag to Makefile

**Files:**
- Modify: `operator/Makefile`

**Step 1: Add feature flag and conditional commands**

Add near the top of Makefile (around line 20):

```makefile
# CSV_PATCHER_IMPL selects which implementation to use for CSV patching
# Valid values: "python" (default), "go"
CSV_PATCHER_IMPL ?= python
```

Then update the bundle target (around line 451):

```makefile
# Run a python script to fix the orders in the specDescriptors (children must not appear before their parents).
ifeq ($(CSV_PATCHER_IMPL),go)
	$(FIX_SPEC_DESCRIPTORS) \
	  <bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
	  >bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed
else
	set -euo pipefail ;\
	$(ACTIVATE_PYTHON) ;\
	bundle_helpers/fix-spec-descriptor-order.py \
	  <bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
	  >bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed
endif
	mv bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed \
       bundle/manifests/rhacs-operator.clusterserviceversion.yaml
```

Update bundle-post-process target (around line 466):

```makefile
.PHONY: bundle-post-process
bundle-post-process: test-bundle-helpers operator-sdk ## Post-process CSV file to include correct operator versions, etc.
ifeq ($(CSV_PATCHER_IMPL),go)
	set -euo pipefail ;\
	first_version=3.62.0 ;\
	candidate_version=$$($(CSV_PATCHER) \
		--use-version $(VERSION) \
		--first-version $${first_version} \
		--operator-image $(IMG) \
		--echo-replaced-version-only \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml); \
	echo "Candidate version: $$candidate_version"; \
	index_img_base=$(INDEX_IMG_BASE); \
	if ! ../scripts/ci/lib.sh check_rhacs_eng_image_exists $${index_img_base##*/} v$${candidate_version}; then \
		echo "Operator index image for this version does not exist (yet)."; \
		unreleased_opt="--unreleased=$${candidate_version}"; \
	else \
		echo "Operator index image for this version exists"; \
	fi; \
	mkdir -p build/ ;\
	rm -rf build/bundle ;\
	cp -a bundle build/ ;\
	cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/ ;\
	$(CSV_PATCHER) \
		--use-version=$(VERSION) \
		--first-version=$${first_version} \
		--operator-image=$(IMG) \
		--related-images-mode=omit \
		$${unreleased_opt:-} \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
		> build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
else
	# Original Python implementation
	set -euo pipefail ;\
	$(ACTIVATE_PYTHON) ;\
	first_version=3.62.0 ;\
	candidate_version=$$(./bundle_helpers/patch-csv.py \
		--use-version $(VERSION) \
		--first-version $${first_version} \
		--operator-image $(IMG) \
		--echo-replaced-version-only \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml); \
	echo "Candidate version: $$candidate_version"; \
	index_img_base=$(INDEX_IMG_BASE); \
	if ! ../scripts/ci/lib.sh check_rhacs_eng_image_exists $${index_img_base##*/} v$${candidate_version}; then \
		echo "Operator index image for this version does not exist (yet)."; \
		unreleased_opt="--unreleased=$${candidate_version}"; \
	else \
		echo "Operator index image for this version exists"; \
	fi; \
	./bundle_helpers/prepare-bundle-manifests.sh \
		--use-version=$(VERSION) \
		--first-version=$${first_version} \
		--operator-image=$(IMG) \
		--related-images-mode=omit \
		$${unreleased_opt:-}
endif
	# Check that the resulting bundle still passes validations.
	$(OPERATOR_SDK) bundle validate ./build/bundle --select-optional suite=operatorframework
```

**Step 2: Update bundle target dependencies**

Update the bundle target to depend on Go tools when using Go implementation:

```makefile
.PHONY: bundle
ifeq ($(CSV_PATCHER_IMPL),go)
bundle: yq manifests kustomize operator-sdk fix-spec-descriptors ## Generate bundle manifests and metadata, then validate generated files.
else
bundle: yq manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
endif
```

Update bundle-post-process dependencies:

```makefile
.PHONY: bundle-post-process
ifeq ($(CSV_PATCHER_IMPL),go)
bundle-post-process: csv-patcher operator-sdk ## Post-process CSV file to include correct operator versions, etc.
else
bundle-post-process: test-bundle-helpers operator-sdk ## Post-process CSV file to include correct operator versions, etc.
endif
```

**Step 3: Test with Python (default)**

```bash
cd operator
make bundle bundle-post-process
```

Expected: Uses Python tools, builds successfully

**Step 4: Test with Go**

```bash
cd operator
CSV_PATCHER_IMPL=go make bundle bundle-post-process
```

Expected: Uses Go tools, builds successfully

**Step 5: Commit**

```bash
git add operator/Makefile
git commit -m "feat(operator): add feature flag for CSV patcher implementation

- Add CSV_PATCHER_IMPL variable (python|go)
- Update bundle target to use Go or Python based on flag
- Update bundle-post-process to use Go or Python
- Default to Python for backwards compatibility"
```

---

## Task 11: Create Validation Test Script

**Files:**
- Create: `operator/test-csv-patcher-equivalence.sh`

**Step 1: Write validation script**

Create `operator/test-csv-patcher-equivalence.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# This script validates that Go and Python CSV patcher implementations produce equivalent output

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building Go tools..."
make csv-patcher fix-spec-descriptors >/dev/null

echo "Building bundle with Python..."
CSV_PATCHER_IMPL=python make bundle bundle-post-process >/dev/null
cp -r build/bundle build/bundle-python

echo "Building bundle with Go..."
rm -rf bundle build/bundle
CSV_PATCHER_IMPL=go make bundle bundle-post-process >/dev/null
cp -r build/bundle build/bundle-go

echo ""
echo "Comparing outputs..."

# Compare CSV files
PYTHON_CSV="build/bundle-python/manifests/rhacs-operator.clusterserviceversion.yaml"
GO_CSV="build/bundle-go/manifests/rhacs-operator.clusterserviceversion.yaml"

if ! diff -u "$PYTHON_CSV" "$GO_CSV"; then
    echo ""
    echo "ERROR: CSV files differ between Python and Go implementations"
    echo "Python: $PYTHON_CSV"
    echo "Go: $GO_CSV"
    echo ""
    echo "Validating both with operator-sdk..."

    # Validate both bundles
    echo "Validating Python bundle..."
    if $(OPERATOR_SDK) bundle validate ./build/bundle-python --select-optional suite=operatorframework; then
        echo "✓ Python bundle is valid"
    else
        echo "✗ Python bundle validation failed"
    fi

    echo ""
    echo "Validating Go bundle..."
    if $(OPERATOR_SDK) bundle validate ./build/bundle-go --select-optional suite=operatorframework; then
        echo "✓ Go bundle is valid"
    else
        echo "✗ Go bundle validation failed"
    fi

    exit 1
fi

echo "✓ CSV files are identical"

# Validate Go bundle
echo ""
echo "Validating Go bundle with operator-sdk..."
if $(OPERATOR_SDK) bundle validate ./build/bundle-go --select-optional suite=operatorframework; then
    echo "✓ Go bundle is valid"
else
    echo "✗ Go bundle validation failed"
    exit 1
fi

echo ""
echo "SUCCESS: Go and Python implementations produce identical, valid output"

# Cleanup
rm -rf build/bundle-python build/bundle-go
```

**Step 2: Make script executable**

```bash
chmod +x operator/test-csv-patcher-equivalence.sh
```

**Step 3: Run validation script**

```bash
cd operator
./test-csv-patcher-equivalence.sh
```

Expected: Either outputs match exactly, or both pass operator-sdk validation

**Step 4: Commit**

```bash
git add operator/test-csv-patcher-equivalence.sh
git commit -m "test(operator): add CSV patcher equivalence validation script

- Compare Python and Go outputs
- Validate both with operator-sdk
- Provide clear success/failure messaging"
```

---

## Task 12: Add Go Unit Tests

**Files:**
- Modify: `operator/Makefile`

**Step 1: Update test-bundle-helpers to run Go tests**

Update the test-bundle-helpers target:

```makefile
.PHONY: test-bundle-helpers
ifeq ($(CSV_PATCHER_IMPL),go)
test-bundle-helpers: ## Run unit tests against CSV patcher tools.
	cd cmd/csv-patcher && go test -v ./...
	cd cmd/fix-spec-descriptors && go test -v ./...
else
test-bundle-helpers: ## Run Python unit tests against helper scripts.
	set -euo pipefail ;\
	$(ACTIVATE_PYTHON) ;\
	pytest -v bundle_helpers
endif
```

**Step 2: Run tests**

```bash
cd operator
CSV_PATCHER_IMPL=go make test-bundle-helpers
```

Expected: All Go tests pass

**Step 3: Commit**

```bash
git add operator/Makefile
git commit -m "test(operator): run Go tests for csv-patcher when using Go impl

- Update test-bundle-helpers to run Go tests when CSV_PATCHER_IMPL=go
- Keep Python tests when using Python implementation"
```

---

## Task 13: Update Konflux Dockerfile

**Files:**
- Modify: `operator/konflux.bundle.Dockerfile`

**Step 1: Replace Python base with Go builder**

Replace `operator/konflux.bundle.Dockerfile`:

```dockerfile
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.25@sha256:527782f4a0270f786192281f68d0374f4a21b3ab759643eee4bfcafb6f539468 AS builder

WORKDIR /stackrox

COPY . .

ARG OPERATOR_IMAGE_TAG
RUN echo "Checking required OPERATOR_IMAGE_TAG"; [[ "${OPERATOR_IMAGE_TAG}" != "" ]]

ARG OPERATOR_IMAGE_REF
RUN echo "Checking required OPERATOR_IMAGE_REF"; [[ "${OPERATOR_IMAGE_REF}" != "" ]]

ARG RELATED_IMAGE_MAIN
ENV RELATED_IMAGE_MAIN=$RELATED_IMAGE_MAIN
RUN echo "Checking required RELATED_IMAGE_MAIN"; [[ "${RELATED_IMAGE_MAIN}" != "" ]]

ARG RELATED_IMAGE_SCANNER
ENV RELATED_IMAGE_SCANNER=$RELATED_IMAGE_SCANNER
RUN echo "Checking required RELATED_IMAGE_SCANNER"; [[ "${RELATED_IMAGE_SCANNER}" != "" ]]

ARG RELATED_IMAGE_SCANNER_DB
ENV RELATED_IMAGE_SCANNER_DB=$RELATED_IMAGE_SCANNER_DB
RUN echo "Checking required RELATED_IMAGE_SCANNER_DB"; [[ "${RELATED_IMAGE_SCANNER_DB}" != "" ]]

ARG RELATED_IMAGE_SCANNER_SLIM
ENV RELATED_IMAGE_SCANNER_SLIM=$RELATED_IMAGE_SCANNER_SLIM
RUN echo "Checking required RELATED_IMAGE_SCANNER_SLIM"; [[ "${RELATED_IMAGE_SCANNER_SLIM}" != "" ]]

ARG RELATED_IMAGE_SCANNER_DB_SLIM
ENV RELATED_IMAGE_SCANNER_DB_SLIM=$RELATED_IMAGE_SCANNER_DB_SLIM
RUN echo "Checking required RELATED_IMAGE_SCANNER_DB_SLIM"; [[ "${RELATED_IMAGE_SCANNER_DB_SLIM}" != "" ]]

ARG RELATED_IMAGE_SCANNER_V4
ENV RELATED_IMAGE_SCANNER_V4=$RELATED_IMAGE_SCANNER_V4
RUN echo "Checking required RELATED_IMAGE_SCANNER_V4"; [[ "${RELATED_IMAGE_SCANNER_V4}" != "" ]]

ARG RELATED_IMAGE_SCANNER_V4_DB
ENV RELATED_IMAGE_SCANNER_V4_DB=$RELATED_IMAGE_SCANNER_V4_DB
RUN echo "Checking required RELATED_IMAGE_SCANNER_V4_DB"; [[ "${RELATED_IMAGE_SCANNER_V4_DB}" != "" ]]

ARG RELATED_IMAGE_COLLECTOR
ENV RELATED_IMAGE_COLLECTOR=$RELATED_IMAGE_COLLECTOR
RUN echo "Checking required RELATED_IMAGE_COLLECTOR"; [[ "${RELATED_IMAGE_COLLECTOR}" != "" ]]

ARG RELATED_IMAGE_ROXCTL
ENV RELATED_IMAGE_ROXCTL=$RELATED_IMAGE_ROXCTL
RUN echo "Checking required RELATED_IMAGE_ROXCTL"; [[ "${RELATED_IMAGE_ROXCTL}" != "" ]]

ARG RELATED_IMAGE_CENTRAL_DB
ENV RELATED_IMAGE_CENTRAL_DB=$RELATED_IMAGE_CENTRAL_DB
RUN echo "Checking required RELATED_IMAGE_CENTRAL_DB"; [[ "${RELATED_IMAGE_CENTRAL_DB}" != "" ]]

# Build Go CSV patcher tools
WORKDIR /stackrox/operator
RUN cd cmd/csv-patcher && go build -o /usr/local/bin/csv-patcher .
RUN cd cmd/fix-spec-descriptors && go build -o /usr/local/bin/fix-spec-descriptors .

# Prepare bundle using Go tools
RUN mkdir -p build/ && \
    rm -rf build/bundle && \
    cp -a bundle build/ && \
    cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/

# Fix descriptor order
RUN /usr/local/bin/fix-spec-descriptors \
      < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
      > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed && \
    mv build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed \
       build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml

# Patch CSV
RUN /usr/local/bin/csv-patcher \
      --use-version="${OPERATOR_IMAGE_TAG}" \
      --first-version=4.0.0 \
      --operator-image="${OPERATOR_IMAGE_REF}" \
      --related-images-mode=konflux \
      < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
      > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml.patched && \
    mv build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml.patched \
       build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml

FROM scratch

ARG OPERATOR_IMAGE_TAG

# Enterprise Contract labels.
LABEL com.redhat.component="rhacs-operator-bundle-container"
LABEL com.redhat.license_terms="https://www.redhat.com/agreements"
LABEL description="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL distribution-scope="public"
LABEL io.k8s.description="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL io.k8s.display-name="operator-bundle"
LABEL io.openshift.tags="rhacs,operator-bundle,stackrox"
LABEL maintainer="Red Hat, Inc."
LABEL name="advanced-cluster-security/rhacs-operator-bundle"
LABEL source-location="https://github.com/stackrox/stackrox"
LABEL summary="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c"
LABEL vendor="Red Hat, Inc."
LABEL version="${OPERATOR_IMAGE_TAG}"
LABEL release="1"

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=rhacs-operator
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-unknown
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Labels for operator certification
LABEL com.redhat.delivery.operator.bundle=true
LABEL com.redhat.openshift.versions="v4.12"

# Use post-processed files
COPY --from=builder /stackrox/operator/build/bundle/manifests /manifests/
COPY --from=builder /stackrox/operator/build/bundle/metadata /metadata/
COPY --from=builder /stackrox/operator/build/bundle/tests/scorecard /tests/scorecard/

COPY LICENSE /licenses/LICENSE
```

**Step 2: Build locally to test**

```bash
cd operator
docker build -f konflux.bundle.Dockerfile \
  --build-arg OPERATOR_IMAGE_TAG=4.0.0-test \
  --build-arg OPERATOR_IMAGE_REF=quay.io/test/operator:4.0.0 \
  --build-arg RELATED_IMAGE_MAIN=quay.io/test/main:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER=quay.io/test/scanner:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER_DB=quay.io/test/scanner-db:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER_SLIM=quay.io/test/scanner-slim:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER_DB_SLIM=quay.io/test/scanner-db-slim:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER_V4=quay.io/test/scanner-v4:4.0.0 \
  --build-arg RELATED_IMAGE_SCANNER_V4_DB=quay.io/test/scanner-v4-db:4.0.0 \
  --build-arg RELATED_IMAGE_COLLECTOR=quay.io/test/collector:4.0.0 \
  --build-arg RELATED_IMAGE_ROXCTL=quay.io/test/roxctl:4.0.0 \
  --build-arg RELATED_IMAGE_CENTRAL_DB=quay.io/test/central-db:4.0.0 \
  -t test-bundle:latest \
  ..
```

Expected: Build succeeds, no Python required

**Step 3: Commit**

```bash
git add operator/konflux.bundle.Dockerfile
git commit -m "feat(operator): replace Python with Go in Konflux bundle build

- Use openshift-golang-builder instead of python-39
- Build csv-patcher and fix-spec-descriptors from source
- Run Go tools directly in Dockerfile
- Remove all Python dependencies from bundle build"
```

---

## Task 14: Switch Default to Go Implementation

**Files:**
- Modify: `operator/Makefile`

**Step 1: Change default CSV_PATCHER_IMPL**

Update line ~20 in Makefile:

```makefile
# CSV_PATCHER_IMPL selects which implementation to use for CSV patching
# Valid values: "python", "go" (default)
CSV_PATCHER_IMPL ?= go
```

**Step 2: Test with new default**

```bash
cd operator
make bundle bundle-post-process
```

Expected: Uses Go tools by default, builds successfully

**Step 3: Test Python still works**

```bash
cd operator
CSV_PATCHER_IMPL=python make bundle bundle-post-process
```

Expected: Python tools still work when explicitly requested

**Step 4: Commit**

```bash
git add operator/Makefile
git commit -m "feat(operator): switch default CSV patcher to Go implementation

- Change CSV_PATCHER_IMPL default from python to go
- Python implementation still available via CSV_PATCHER_IMPL=python
- Go is now the primary implementation"
```

---

## Task 15: Remove Python Code and Dependencies

**Files:**
- Delete: `operator/bundle_helpers/*.py`
- Delete: `operator/bundle_helpers/requirements*.txt`
- Delete: `operator/bundle_helpers/prepare-bundle-manifests.sh`
- Modify: `operator/Makefile` (remove Python-related code)
- Modify: `operator/.gitignore` (remove .venv)

**Step 1: Remove Python files**

```bash
cd operator
git rm bundle_helpers/*.py
git rm bundle_helpers/requirements*.txt
git rm bundle_helpers/prepare-bundle-manifests.sh
```

**Step 2: Remove Python from Makefile**

Remove these sections from `operator/Makefile`:

- `PYTHON ?= python3` variable (line ~17)
- `ACTIVATE_PYTHON` variable (lines ~420-423)
- All Python-related conditionals in bundle and bundle-post-process targets
- Keep only Go implementation code

Updated bundle target:

```makefile
.PHONY: bundle
bundle: yq manifests kustomize operator-sdk fix-spec-descriptors ## Generate bundle manifests and metadata, then validate generated files.
	rm -rf bundle
	rm -rf config/manifests/bases && $(OPERATOR_SDK) generate kustomize manifests --input-dir=config/ui-metadata
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/stackrox-io/stackrox-operator:0.0.1
	cd config/scorecard-versioned && $(KUSTOMIZE) edit set image quay.io/operator-framework/scorecard-test=quay.io/operator-framework/scorecard-test:$(OPERATOR_SDK_VERSION)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(YQ) -i '.metadata.annotations.createdAt = ""' bundle/manifests/rhacs-operator.clusterserviceversion.yaml
	sed -i'.bak' -e '/operators\.operatorframework\.io\.bundle\.channel/d' bundle.Dockerfile
	sed -i'.bak' -e '/# Copy files to locations specified by labels./d' bundle.Dockerfile
	sed -i'.bak' -E -e '/^COPY .* \/(manifests|metadata|tests\/scorecard)\/$$/d' bundle.Dockerfile
	rm -f bundle.Dockerfile.bak
	cat bundle.Dockerfile.extra >> bundle.Dockerfile
	$(FIX_SPEC_DESCRIPTORS) \
	  <bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
	  >bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed
	mv bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed \
       bundle/manifests/rhacs-operator.clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle --select-optional suite=operatorframework
```

Updated bundle-post-process target:

```makefile
.PHONY: bundle-post-process
bundle-post-process: csv-patcher operator-sdk ## Post-process CSV file to include correct operator versions, etc.
	set -euo pipefail ;\
	first_version=3.62.0 ;\
	candidate_version=$$($(CSV_PATCHER) \
		--use-version $(VERSION) \
		--first-version $${first_version} \
		--operator-image $(IMG) \
		--echo-replaced-version-only \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml); \
	echo "Candidate version: $$candidate_version"; \
	index_img_base=$(INDEX_IMG_BASE); \
	if ! ../scripts/ci/lib.sh check_rhacs_eng_image_exists $${index_img_base##*/} v$${candidate_version}; then \
		echo "Operator index image for this version does not exist (yet)."; \
		unreleased_opt="--unreleased=$${candidate_version}"; \
	else \
		echo "Operator index image for this version exists"; \
	fi; \
	mkdir -p build/ ;\
	rm -rf build/bundle ;\
	cp -a bundle build/ ;\
	cp -v ../config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml build/bundle/manifests/ ;\
	$(CSV_PATCHER) \
		--use-version=$(VERSION) \
		--first-version=$${first_version} \
		--operator-image=$(IMG) \
		--related-images-mode=omit \
		$${unreleased_opt:-} \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
		> build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./build/bundle --select-optional suite=operatorframework
```

Updated test-bundle-helpers target:

```makefile
.PHONY: test-bundle-helpers
test-bundle-helpers: ## Run unit tests against CSV patcher tools.
	cd cmd/csv-patcher && go test -v ./...
	cd cmd/fix-spec-descriptors && go test -v ./...
```

**Step 3: Remove .venv from .gitignore if present**

```bash
cd operator
# Check if bundle_helpers/.venv is in .gitignore
grep -v "bundle_helpers/.venv" .gitignore > .gitignore.new || true
mv .gitignore.new .gitignore
```

**Step 4: Test build still works**

```bash
cd operator
make clean
make bundle bundle-post-process test-bundle-helpers
```

Expected: Everything builds successfully using only Go tools

**Step 5: Commit**

```bash
git add -A
git commit -m "feat(operator): remove Python code and dependencies

- Delete all Python scripts from bundle_helpers/
- Delete Python requirements files
- Delete prepare-bundle-manifests.sh wrapper
- Remove PYTHON and ACTIVATE_PYTHON from Makefile
- Remove all Python conditionals from Makefile
- Operator build now uses only Go tools
- Python dependency fully eliminated"
```

---

## Task 16: Update Documentation

**Files:**
- Create: `operator/docs/csv-patching.md`
- Modify: `operator/README.md` (if exists and mentions Python)

**Step 1: Create CSV patching documentation**

Create `operator/docs/csv-patching.md`:

```markdown
# CSV Patching Tools

The operator bundle build process uses two Go CLI tools to patch the ClusterServiceVersion (CSV) YAML file:

## Tools

### csv-patcher

Patches the CSV with version information, operator images, and related images.

**Location:** `operator/cmd/csv-patcher/`

**Usage:**
```bash
csv-patcher \
  --use-version=4.0.0 \
  --first-version=3.62.0 \
  --operator-image=quay.io/stackrox-io/stackrox-operator:4.0.0 \
  --related-images-mode=omit \
  < input.yaml \
  > output.yaml
```

**Flags:**
- `--use-version` - Version to set in CSV (e.g., "4.0.0")
- `--first-version` - First operator version ever released (e.g., "3.62.0")
- `--operator-image` - Operator container image reference
- `--related-images-mode` - How to handle related images: `downstream`, `omit`, `konflux`
- `--add-supported-arch` - Comma-separated list of supported architectures (default: amd64,arm64,ppc64le,s390x)
- `--echo-replaced-version-only` - Only calculate and print the replaced version
- `--unreleased` - Version that is not yet released (used for version calculation)

**What it does:**
- Updates `metadata.name` with version (e.g., `rhacs-operator.v4.0.0`)
- Updates `spec.version` field
- Replaces placeholder operator image with actual image
- Calculates and sets `spec.replaces` (which previous version this replaces)
- Sets `olm.skipRange` annotation for upgrade paths
- Adds multi-architecture labels
- Handles related images based on mode
- Adds SecurityPolicy CRD to owned CRDs
- Updates `createdAt` timestamp

**Version Calculation Logic:**
The tool implements complex logic to determine which version this release replaces:
- First release gets no `replaces` field
- Patch releases replace previous patch (e.g., 4.0.2 replaces 4.0.1)
- Y-Stream releases replace previous Y-Stream (e.g., 4.1.0 replaces 4.0.0)
- Handles version skips (broken versions that should be skipped)
- Handles unreleased versions

### fix-spec-descriptors

Fixes the ordering of spec descriptors in the CSV so that parent paths always appear before children.

**Location:** `operator/cmd/fix-spec-descriptors/`

**Usage:**
```bash
fix-spec-descriptors < input.yaml > output.yaml
```

**What it does:**
- Sorts `specDescriptors` so parent paths come before child paths
- Resolves relative field dependencies to absolute paths
- Ensures CSV validation passes

## Build Integration

The tools are built automatically when needed:

```bash
make csv-patcher          # Build csv-patcher
make fix-spec-descriptors # Build fix-spec-descriptors
```

They are used in the bundle build process:

```bash
make bundle              # Uses fix-spec-descriptors
make bundle-post-process # Uses csv-patcher
```

## Testing

Run unit tests:

```bash
make test-bundle-helpers
```

This runs Go tests for both tools.

## Migration from Python

These Go tools replace the previous Python-based implementation:
- `bundle_helpers/patch-csv.py` → `cmd/csv-patcher/`
- `bundle_helpers/fix-spec-descriptor-order.py` → `cmd/fix-spec-descriptors/`
- `bundle_helpers/prepare-bundle-manifests.sh` → Inlined into Makefile

The Go implementation provides:
- ✅ No Python dependency in operator build
- ✅ Faster build times
- ✅ Better type safety
- ✅ Easier testing with standard Go tooling
- ✅ Simpler CI/CD pipeline

## Development

### Adding New Fields to Patch

To patch additional CSV fields:

1. Update `csv.go` if new struct fields are needed
2. Add patching logic to `patch.go` in the `PatchCSV` function
3. Add test cases to `patch_test.go`
4. Run tests: `go test ./...`

### Modifying Version Calculation

Version calculation logic is in `version.go`:
- `GetPreviousYStream()` - Calculates previous Y-Stream version
- `CalculateReplacedVersion()` - Complex logic for determining replaces version

Changes to this logic should include comprehensive test coverage.
```

**Step 2: Update main operator README if needed**

If `operator/README.md` exists and mentions Python:

```bash
cd operator
# Check for Python mentions
grep -i python README.md
```

If found, update to mention Go tools instead.

**Step 3: Commit**

```bash
git add operator/docs/csv-patching.md operator/README.md
git commit -m "docs(operator): add CSV patching tools documentation

- Document csv-patcher and fix-spec-descriptors tools
- Explain usage, flags, and what each tool does
- Document version calculation logic
- Note migration from Python
- Add development guidelines"
```

---

## Task 17: Final Validation and Testing

**Files:**
- None (validation only)

**Step 1: Clean build from scratch**

```bash
cd operator
make clean
rm -rf bin/ build/ bundle/
```

**Step 2: Full build with Go tools**

```bash
make bundle bundle-post-process
```

Expected: Builds successfully, no errors

**Step 3: Run all tests**

```bash
make test-bundle-helpers
```

Expected: All Go tests pass

**Step 4: Validate bundle**

```bash
make operator-sdk
$(OPERATOR_SDK) bundle validate ./build/bundle --select-optional suite=operatorframework
```

Expected: Bundle validation passes

**Step 5: Build bundle image locally**

```bash
make bundle-build
```

Expected: Bundle image builds successfully

**Step 6: Run scorecard tests**

```bash
make bundle-test
```

Expected: Scorecard tests pass

**Step 7: Document results**

Create validation summary in commit message.

**Step 8: Commit validation results**

```bash
git add -A
git commit -m "test(operator): validate Go CSV patcher implementation

Validation results:
- ✅ Bundle builds successfully
- ✅ All unit tests pass
- ✅ operator-sdk bundle validate passes
- ✅ Bundle image builds
- ✅ Scorecard tests pass

The Go implementation is fully functional and ready for production use.
Python dependency has been completely removed from operator build."
```

---

## Completion Checklist

- [x] Task 1: XyzVersion type and parsing
- [x] Task 2: Previous Y-Stream calculation
- [x] Task 3: Replace version calculation
- [x] Task 4: CSV structure types
- [x] Task 5: String replacement utility
- [x] Task 6: Main CSV patching logic
- [x] Task 7: csv-patcher CLI entry point
- [x] Task 8: fix-spec-descriptors CLI tool
- [x] Task 9: Makefile build targets
- [x] Task 10: Feature flag implementation
- [x] Task 11: Validation test script
- [x] Task 12: Go unit tests in Makefile
- [x] Task 13: Update Konflux Dockerfile
- [x] Task 14: Switch default to Go
- [x] Task 15: Remove Python code
- [x] Task 16: Update documentation
- [x] Task 17: Final validation

## Success Criteria

✅ All Python code removed from `operator/bundle_helpers/`
✅ No Python dependencies in operator build
✅ Go tools produce functionally equivalent output to Python tools
✅ All unit tests pass
✅ operator-sdk bundle validation passes
✅ Konflux bundle build uses only Go tools
✅ Documentation complete

## Notes

- The implementation preserves exact Python logic for version calculation
- Bundle outputs are validated with operator-sdk, ensuring functional equivalence
- Byte-identical output is ideal but not required if both pass validation
- Feature flag allows safe testing and easy rollback during transition
- Clean commit history with atomic changes and clear messages

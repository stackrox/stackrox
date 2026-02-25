package csv

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/rewrite"
	"github.com/stackrox/rox/operator/bundle_helpers/pkg/values"
	"helm.sh/helm/v3/pkg/chartutil"
)

// PatchOptions contains all options for patching a CSV
type PatchOptions struct {
	Version             string
	OperatorImage       string
	FirstVersion        string
	RelatedImagesMode   string
	ExtraSupportedArchs []string
	Unreleased          string
}

// PatchCSV modifies the CSV document in-place according to options
func PatchCSV(doc chartutil.Values, opts PatchOptions) error {
	// Update createdAt timestamp
	if err := values.SetValue(doc, "metadata.annotations.createdAt", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("failed to set createdAt: %w", err)
	}

	// Replace placeholder image with actual operator image
	placeholderImage, err := values.GetString(doc, "metadata.annotations.containerImage")
	if err != nil {
		return fmt.Errorf("failed to get containerImage: %w", err)
	}
	rewrite.Strings(doc, placeholderImage, opts.OperatorImage)

	// Update metadata name with version
	metadataName, err := values.GetString(doc, "metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get metadata.name: %w", err)
	}
	rawName := strings.TrimSuffix(metadataName, ".v0.0.1")
	if !strings.HasSuffix(metadataName, ".v0.0.1") {
		return fmt.Errorf("metadata.name does not end with .v0.0.1: %s", metadataName)
	}
	if err := values.SetValue(doc, "metadata.name", fmt.Sprintf("%s.v%s", rawName, opts.Version)); err != nil {
		return fmt.Errorf("failed to set metadata.name: %w", err)
	}

	// Update spec.version
	if err := values.SetValue(doc, "spec.version", opts.Version); err != nil {
		return fmt.Errorf("failed to set spec.version: %w", err)
	}
	spec, err := values.GetMap(doc, "spec")
	if err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}

	// Handle related images based on mode
	if opts.RelatedImagesMode != "omit" {
		if err := injectRelatedImageEnvVars(spec); err != nil {
			return err
		}
	}

	switch opts.RelatedImagesMode {
	case "downstream":
		delete(spec, "relatedImages")
	case "omit":
		delete(spec, "relatedImages")
	case "konflux":
		if err := constructRelatedImages(spec, opts.OperatorImage); err != nil {
			return err
		}
	}

	// Add multi-arch labels (label keys contain dots, so access the map directly)
	if len(opts.ExtraSupportedArchs) > 0 {
		metadata, err := values.GetMap(doc, "metadata")
		if err != nil {
			return fmt.Errorf("failed to get metadata: %w", err)
		}
		if _, exists := metadata["labels"]; !exists {
			metadata["labels"] = map[string]any{}
		}
		labels, err := values.GetMap(metadata, "labels")
		if err != nil {
			return fmt.Errorf("failed to get metadata.labels: %w", err)
		}
		for _, arch := range opts.ExtraSupportedArchs {
			labels[fmt.Sprintf("operatorframework.io/arch.%s", arch)] = "supported"
		}
	}

	// Calculate previous Y-stream and replaced version
	previousYStream, replacedVersion, err := CalculateReplacedVersionForCSV(
		opts.Version,
		opts.FirstVersion,
		opts.Unreleased,
		rawName,
		spec,
	)
	if err != nil {
		return err
	}

	// Set olm.skipRange (annotation key contains a dot, so access the map directly)
	annotations, err := values.GetMap(doc, "metadata.annotations")
	if err != nil {
		return fmt.Errorf("failed to get metadata.annotations: %w", err)
	}
	annotations["olm.skipRange"] = fmt.Sprintf(">= %s < %s", previousYStream, opts.Version)

	// Only set replaces if there is a replacement version
	if replacedVersion != nil {
		if err := values.SetValue(doc, "spec.replaces", fmt.Sprintf("%s.v%s", rawName, replacedVersion.String())); err != nil {
			return fmt.Errorf("failed to set spec.replaces: %w", err)
		}
	}

	// Improve SecurityPolicy CRD metadata in ACS operator CSV
	if err := addSecurityPolicyCRD(spec); err != nil {
		return err
	}

	return nil
}

func injectRelatedImageEnvVars(spec chartutil.Values) error {
	// Find all RELATED_IMAGE_* env vars in the spec and replace with actual values
	var traverse func(any) error
	traverse = func(data any) error {
		switch v := data.(type) {
		case map[string]any:
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
		case chartutil.Values:
			// chartutil.Values is a named type over map[string]interface{}; convert
			// and re-traverse so the map[string]any branch handles it uniformly.
			return traverse(map[string]any(v))
		case []any:
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

func constructRelatedImages(spec chartutil.Values, managerImage string) error {
	if managerImage == "" {
		return errors.New("managerImage cannot be empty")
	}

	relatedImages := make([]map[string]any, 0)

	// Collect all RELATED_IMAGE_* env vars
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "RELATED_IMAGE_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("malformed RELATED_IMAGE environment variable: %s", envVar)
			}
			name := strings.TrimPrefix(parts[0], "RELATED_IMAGE_")
			name = strings.ToLower(name)
			image := parts[1]

			relatedImages = append(relatedImages, map[string]any{
				"name":  name,
				"image": image,
			})
		}
	}

	// Add manager image
	relatedImages = append(relatedImages, map[string]any{
		"name":  "manager",
		"image": managerImage,
	})

	spec["relatedImages"] = relatedImages
	return nil
}

func addSecurityPolicyCRD(spec chartutil.Values) error {
	crd := map[string]any{
		"name":        "securitypolicies.config.stackrox.io",
		"version":     "v1alpha1",
		"kind":        "SecurityPolicy",
		"displayName": "Security Policy",
		"description": "SecurityPolicy is the schema for the policies API.",
		"resources": []map[string]any{
			{
				"kind":    "Deployment",
				"name":    "",
				"version": "v1",
			},
		},
	}

	crdsVal := spec["customresourcedefinitions"]
	if crdsVal == nil {
		return errors.New("spec.customresourcedefinitions field is missing")
	}

	var crds map[string]any
	switch v := crdsVal.(type) {
	case map[string]any:
		crds = v
	case chartutil.Values:
		crds = v
	default:
		return fmt.Errorf("spec.customresourcedefinitions has wrong type: %T", crdsVal)
	}

	owned, ok := crds["owned"].([]any)
	if !ok {
		return errors.New("spec.customresourcedefinitions.owned field is missing or has wrong type")
	}

	// Filter out existing SecurityPolicy CRDs to prevent duplicates
	filteredOwned := make([]any, 0, len(owned))
	for _, crdEntry := range owned {
		var crdMap map[string]any
		switch v := crdEntry.(type) {
		case map[string]any:
			crdMap = v
		case chartutil.Values:
			crdMap = v
		default:
			filteredOwned = append(filteredOwned, crdEntry)
			continue
		}
		kind, _ := crdMap["kind"].(string)
		if kind != "SecurityPolicy" {
			filteredOwned = append(filteredOwned, crdEntry)
		}
	}

	// Add the SecurityPolicy CRD
	crds["owned"] = append(filteredOwned, crd)

	return nil
}

// ProcessSkips extracts and parses skip versions from the spec.
// It filters out the placeholder "0.0.1" version and returns parsed XyzVersion entries.
// The prefix parameter should be the operator name prefix (e.g., "rhacs-operator").
func ProcessSkips(prefix string, spec chartutil.Values) ([]XyzVersion, error) {
	skips := make([]XyzVersion, 0)
	if rawSkips, ok := spec["skips"].([]any); ok {
		for _, s := range rawSkips {
			skipStr, ok := s.(string)
			if !ok {
				return nil, errors.New("skip entry is not a string")
			}
			// Filter out the placeholder version
			if skipStr == prefix+".v0.0.1" {
				continue
			}
			// Extract version from prefix
			skipVer := strings.TrimPrefix(skipStr, prefix+".")

			v, err := ParseXyzVersion(skipVer)
			if err != nil {
				return nil, err
			}
			skips = append(skips, v)
		}
	}
	return skips, nil
}

// CalculateReplacedVersionForCSV encapsulates the common logic for calculating
// the replaced version from a CSV document. It returns both the previous
// Y-stream version (used for skipRange) and the calculated replaced version.
func CalculateReplacedVersionForCSV(
	version, firstVersion, unreleased string,
	operatorNamePrefix string,
	spec chartutil.Values,
) (previousYStream string, replacedVersion *XyzVersion, err error) {
	// Parse version strings early
	versionXyz, err := ParseXyzVersion(version)
	if err != nil {
		return "", nil, err
	}

	firstXyz, err := ParseXyzVersion(firstVersion)
	if err != nil {
		return "", nil, err
	}

	// Parse skips
	skips, err := ProcessSkips(operatorNamePrefix, spec)
	if err != nil {
		return "", nil, err
	}

	// Calculate previous Y-Stream (still returns string for skipRange)
	previousYStream, err = GetPreviousYStream(version)
	if err != nil {
		return "", nil, err
	}

	// Parse previousYStream once
	previousXyz, err := ParseXyzVersion(previousYStream)
	if err != nil {
		return "", nil, err
	}

	// Calculate replaced version with parsed XyzVersion values
	replacedVersion, err = CalculateReplacedVersion(
		versionXyz,
		firstXyz,
		previousXyz,
		skips,
		unreleased,
	)
	if err != nil {
		return "", nil, err
	}

	return previousYStream, replacedVersion, nil
}

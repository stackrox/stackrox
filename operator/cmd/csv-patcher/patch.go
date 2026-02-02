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

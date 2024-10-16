package customresource

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

var (
	invalidCRDNameCharPattern = regexp.MustCompile(`[^a-z0-9\.\-]`)
	consecutivePattern        = regexp.MustCompile(`[.-]{2,}`)
)

// CustomResource represents the structure for a SecurityPolicy Kubernetes Custom Resource.
type CustomResource struct {
	APIVersion         string                 `yaml:"apiVersion"`
	Kind               string                 `yaml:"kind"`
	Metadata           map[string]interface{} `yaml:"metadata,omitempty"`
	SecurityPolicySpec *Policy                `yaml:"spec"`
}

// GenerateCustomResource generate custom resource in YAML text from a policy
func GenerateCustomResource(policy *storage.Policy) (string, error) {
	convertedPolicy := ConvertPolicyToCustomResource(policy)
	w := &bytes.Buffer{}
	enc := yaml.NewEncoder(w)
	defer utils.IgnoreError(enc.Close)
	// Use idiomatic indentation.
	enc.SetIndent(2)
	if err := enc.Encode(convertedPolicy); err != nil {
		return "", err
	}
	return w.String(), nil
}

func timestampToFormatRFC3339(ts *timestamppb.Timestamp) string {
	t := protocompat.ConvertTimestampToTimeOrNil(ts)
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// toDNSSubdomainName converts a policy name to a valid DNS subdomain name.
func toDNSSubdomainName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	// Replace all invalid characters with '-'
	name = invalidCRDNameCharPattern.ReplaceAllString(name, "-")

	// Remove consecutive dots or hyphens
	name = consecutivePattern.ReplaceAllString(name, "-")

	// Truncate to 253 characters max, as per DNS subdomain name requirements
	if len(name) > 253 {
		name = name[:253]
	}

	name = strings.Trim(name, "-.")

	// If the result is empty or invalid after processing, return a default value
	if len(name) == 0 {
		// Get a random alphanumeric string from the first segment of a uuid
		randAlphaNumerics := strings.Split(uuid.NewV4().String(), "-")
		if len(randAlphaNumerics) > 0 && len(randAlphaNumerics[0]) > 0 {
			name = fmt.Sprintf("rhacs-%s", randAlphaNumerics[0])
		} else {
			// This should not happen but we just add this as a proof for codes from external package.
			utils.Should(errors.Errorf("Unexpected error: failed to create a name for policy from uuid: %+v", randAlphaNumerics))
			return "rhacs-default-policy"
		}
	}
	return name
}

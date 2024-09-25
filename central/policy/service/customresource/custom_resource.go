package customresource

import (
	"bytes"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

// CustomResource represents the structure for a SecurityPolicy Kubernetes Custom Resource.
type CustomResource struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"`
	Spec       *Policy                `yaml:"spec"`
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

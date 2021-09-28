package converter

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// ComposeID creates an active component id from a deployment id and a component id
func ComposeID(deploymentID, componentID string) string {
	return fmt.Sprintf("%s:%s", deploymentID, componentID)
}

// DecomposeID splits an active component id to a deployment id and a component id
func DecomposeID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid active component id: %q", id)
	}
	return parts[0], parts[1], nil
}

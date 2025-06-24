package containers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Assert that container detection is running correctly by checking if it returns true in CI. Expected to return
// false when run locally.
func TestContainerDetection(t *testing.T) {
	_, runningInGithub := os.LookupEnv("GITHUB_ACTIONS")

	// How to determine I'm runing in k8s besides what is already done in IsRunningInContainer()?
	// Potentially check for a KUBERNETES_* env var?
	_, err := os.Stat("/run/secrets/kubernetes.io/serviceaccount/namespace")
	runningInK8s := err == nil // File that only exists in k8s pods is found
	if runningInGithub || runningInK8s {
		assert.True(t, IsRunningInContainer())
	} else {
		assert.False(t, IsRunningInContainer())
	}
}

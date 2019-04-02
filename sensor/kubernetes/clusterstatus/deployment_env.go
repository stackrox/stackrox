package clusterstatus

import (
	"fmt"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// getDeploymentEnvironment extracts a "deployment environment" (such as "docker-for-desktop" or "gcp/<project>") from a
// node.
// NOTE: This is only used for license enforcement, and further only for development/CI/QA/demo/... licenses to make
// them more restricted. As such, we only need to extract those deployment environments that we use/care about.
// Since this is not surfaced to the user, it is fine if we return "unknown" here for most customer deployments, as we
// do not anticipate issuing deployment environment-restricted licenses to customers. If somebody manages to obtain
// one of our internal-only licenses, however, a deployment environment of "unknown" will most likely cause the license
// to be rejected, which is intended.
func getDeploymentEnvironment(node *v1.Node) string {
	if node == nil {
		return ""
	}

	if node.Spec.ProviderID != "" {
		if strings.HasPrefix(node.Spec.ProviderID, "gce://") {
			components := strings.SplitN(node.Spec.ProviderID[6:], "/", 2)
			return fmt.Sprintf("gcp/%s", components[0])
		}
	}
	if node.Spec.ExternalID == "docker-for-desktop" {
		return "docker-for-desktop"
	}
	return "unknown"
}

type deploymentEnvSet struct {
	envsAndCount map[string]int
}

func newDeploymentEnvSet() *deploymentEnvSet {
	return &deploymentEnvSet{
		envsAndCount: make(map[string]int),
	}
}

func (s *deploymentEnvSet) Add(env string) bool {
	if env == "" {
		return false
	}
	cnt := s.envsAndCount[env]
	s.envsAndCount[env] = cnt + 1

	return cnt == 0
}

func (s *deploymentEnvSet) Replace(new, old string) bool {
	if new == old {
		return false
	}

	changed := s.Remove(old)
	changed = s.Add(new) || changed
	return changed
}

func (s *deploymentEnvSet) Remove(env string) bool {
	if env == "" {
		return false
	}
	cnt := s.envsAndCount[env]
	cnt--
	if cnt == 0 {
		delete(s.envsAndCount, env)
		return true
	}
	s.envsAndCount[env] = cnt
	return false
}

func (s *deploymentEnvSet) AsSlice() []string {
	result := make([]string, 0, len(s.envsAndCount))

	for env := range s.envsAndCount {
		result = append(result, env)
	}

	sort.Strings(result)
	return result
}

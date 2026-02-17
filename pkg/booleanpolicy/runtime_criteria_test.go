package booleanpolicy

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRuntimeCriteria(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	suite.Run(t, new(RuntimeCriteriaTestSuite))
}

type RuntimeCriteriaTestSuite struct {
	basePoliciesTestSuite
}

func processBaselineMessage(dep *storage.Deployment, baseline bool, privileged bool, processNames ...string) []*storage.Alert_Violation {
	violations := make([]*storage.Alert_Violation, 0, len(processNames))
	containerName := dep.GetContainers()[0].GetName()
	for _, p := range processNames {
		if baseline {
			msg := fmt.Sprintf("Unexpected process '%s' in container '%s'", p, containerName)
			violations = append(violations, &storage.Alert_Violation{Message: msg})
		}
		if privileged {
			violations = append(violations, privilegedMessage(dep)...)
		}
	}
	return violations
}

func privilegedMessage(dep *storage.Deployment) []*storage.Alert_Violation {
	containerName := dep.GetContainers()[0].GetName()
	return []*storage.Alert_Violation{{Message: fmt.Sprintf("Container '%s' is privileged", containerName)}}
}

func newIndicator(deployment *storage.Deployment, name, args, execFilePath string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		ContainerName: deployment.GetContainers()[0].GetName(),
		Signal: &storage.ProcessSignal{
			Name:         name,
			Args:         args,
			ExecFilePath: execFilePath,
		},
	}
}

func podExecViolationMsg(pod, container, command string) *storage.Alert_Violation {
	if command == "" {
		return &storage.Alert_Violation{
			Message: fmt.Sprintf("Kubernetes API received exec request into pod '%s' container '%s'", pod, container),
			Type:    storage.Alert_Violation_K8S_EVENT,
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
						{Key: "pod", Value: pod},
						{Key: "container", Value: container},
					},
				},
			},
		}
	}

	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubernetes API received exec '%s' request into pod '%s' container '%s'",
			command, pod, container),
		Type: storage.Alert_Violation_K8S_EVENT,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "container", Value: container},
					{Key: "commands", Value: command},
				},
			},
		},
	}
}

func podPortForwardViolationMsg(pod string, port int) *storage.Alert_Violation {
	return &storage.Alert_Violation{
		Message: fmt.Sprintf("Kubernetes API received port forward request to pod '%s' ports '%s'", pod, strconv.Itoa(port)),
		Type:    storage.Alert_Violation_K8S_EVENT,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
					{Key: "pod", Value: pod},
					{Key: "ports", Value: strconv.Itoa(port)},
				},
			},
		},
	}
}

func podExecEvent(pod, container, command string) *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Object: &storage.KubernetesEvent_Object{
			Name:     pod,
			Resource: storage.KubernetesEvent_Object_PODS_EXEC,
		},
		ObjectArgs: &storage.KubernetesEvent_PodExecArgs_{
			PodExecArgs: &storage.KubernetesEvent_PodExecArgs{
				Container: container,
				Commands:  []string{command},
			},
		},
	}
}

func podPortForwardEvent(pod string, port int32) *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Object: &storage.KubernetesEvent_Object{
			Name:     pod,
			Resource: storage.KubernetesEvent_Object_PODS_PORTFORWARD,
		},
		ObjectArgs: &storage.KubernetesEvent_PodPortForwardArgs_{
			PodPortForwardArgs: &storage.KubernetesEvent_PodPortForwardArgs{
				Ports: []int32{port},
			},
		},
	}
}

func podAttachEvent(pod, container string) *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Object: &storage.KubernetesEvent_Object{
			Name:     pod,
			Resource: storage.KubernetesEvent_Object_PODS_ATTACH,
		},
		ObjectArgs: &storage.KubernetesEvent_PodAttachArgs_{
			PodAttachArgs: &storage.KubernetesEvent_PodAttachArgs{
				Container: container,
			},
		},
	}
}

func podAttachViolationMsg(pod, container string) *storage.Alert_Violation {
	attrs := []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
		{Key: "pod", Value: pod},
	}
	if container != "" {
		attrs = append(attrs, &storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{Key: "container", Value: container})
	}

	message := "Kubernetes API received attach request"
	if pod != "" {
		message = fmt.Sprintf("Kubernetes API received attach request to pod '%s'", pod)
		if container != "" {
			message = fmt.Sprintf("Kubernetes API received attach request to pod '%s' container '%s'", pod, container)
		}
	}

	return &storage.Alert_Violation{
		Message: message,
		Type:    storage.Alert_Violation_K8S_EVENT,
		MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
			KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
				Attrs: attrs,
			},
		},
	}
}

func (suite *RuntimeCriteriaTestSuite) TestProcessBaseline() {
	privilegedDep := fixtures.GetDeployment().CloneVT()
	privilegedDep.Id = "PRIVILEGED"
	suite.addDepAndImages(privilegedDep)

	nonPrivilegedDep := fixtures.GetDeployment().CloneVT()
	nonPrivilegedDep.Id = "NOTPRIVILEGED"
	nonPrivilegedDep.Containers[0].SecurityContext.Privileged = false
	suite.addDepAndImages(nonPrivilegedDep)

	const aptGetKey = "apt-get"
	const aptGet2Key = "apt-get2"
	const curlKey = "curl"
	const bashKey = "bash"

	indicators := make(map[string]map[string]*storage.ProcessIndicator)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		indicators[dep.GetId()] = map[string]*storage.ProcessIndicator{
			aptGetKey:  suite.addIndicator(dep.GetId(), "apt-get", "install nginx", "/bin/apt-get", nil, 0),
			aptGet2Key: suite.addIndicator(dep.GetId(), "apt-get", "update", "/bin/apt-get", nil, 0),
			curlKey:    suite.addIndicator(dep.GetId(), "curl", "https://stackrox.io", "/bin/curl", nil, 0),
			bashKey:    suite.addIndicator(dep.GetId(), "bash", "attach.sh", "/bin/bash", nil, 0),
		}
	}
	processesNotInBaseline := map[string]set.StringSet{
		privilegedDep.GetId():    set.NewStringSet(aptGetKey, aptGet2Key, bashKey),
		nonPrivilegedDep.GetId(): set.NewStringSet(aptGetKey, curlKey, bashKey),
	}

	// Plain groups
	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)
	privilegedGroup := policyGroupWithSingleKeyValue(fieldnames.PrivilegedContainer, "true", false)
	baselineGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedProcessExecuted, "true", false)

	for _, testCase := range []struct {
		groups []*storage.PolicyGroup

		// Deployment ids to indicator keys
		expectedMatches        map[string][]string
		expectedProcessMatches map[string][]string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			groups: []*storage.PolicyGroup{aptGetGroup},
			// only process violation, no alert violation
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups:          []*storage.PolicyGroup{baselineGroup},
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
		},

		{
			groups: []*storage.PolicyGroup{privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get", "curl", "bash"),
			},
		},
		{
			groups:          []*storage.PolicyGroup{aptGetGroup, baselineGroup},
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
	} {
		c := testCase
		suite.T().Run(fmt.Sprintf("%+v", c.groups), func(t *testing.T) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)

			m, err := BuildDeploymentWithProcessMatcher(policy)
			require.NoError(t, err)

			actualMatches := make(map[string][]string)
			actualProcessMatches := make(map[string][]string)
			actualViolations := make(map[string][]*storage.Alert_Violation)
			for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
				for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
					violations, err := m.MatchDeploymentWithProcess(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)), indicators[dep.GetId()][key], processesNotInBaseline[dep.GetId()].Contains(key))
					suite.Require().NoError(err)
					if len(violations.AlertViolations) > 0 {
						actualMatches[dep.GetId()] = append(actualMatches[dep.GetId()], key)
						actualViolations[dep.GetId()] = append(actualViolations[dep.GetId()], violations.AlertViolations...)
					}
					if violations.ProcessViolation != nil {
						actualProcessMatches[dep.GetId()] = append(actualProcessMatches[dep.GetId()], key)
					}

				}
			}
			assert.Equal(t, c.expectedMatches, actualMatches)
			assert.Equal(t, c.expectedProcessMatches, actualProcessMatches)

			for id, violations := range c.expectedViolations {
				assert.Contains(t, actualViolations, id)
				protoassert.ElementsMatch(t, violations, actualViolations[id])
			}
		})
	}
}

func (suite *RuntimeCriteriaTestSuite) TestKubeEventConstraints() {
	podExecGroup := policyGroupWithSingleKeyValue(fieldnames.KubeResource, "PODS_EXEC", false)
	podAttachGroup := policyGroupWithSingleKeyValue(fieldnames.KubeResource, "PODS_ATTACH", false)

	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)

	for _, c := range []struct {
		event              *storage.KubernetesEvent
		groups             []*storage.PolicyGroup
		expectedViolations []*storage.Alert_Violation
		builderErr         bool
		withProcessSection bool
	}{
		// PODS_EXEC test cases
		{
			event:              podExecEvent("p1", "c1", "cmd"),
			groups:             []*storage.PolicyGroup{podExecGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "cmd")},
		},
		{
			event:              podExecEvent("p1", "c1", ""),
			groups:             []*storage.PolicyGroup{podExecGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
		},

		{
			groups: []*storage.PolicyGroup{podExecGroup},
		},
		{
			event:  podPortForwardEvent("p1", 8000),
			groups: []*storage.PolicyGroup{podExecGroup},
		},
		{
			event:      podPortForwardEvent("p1", 8000),
			groups:     []*storage.PolicyGroup{podExecGroup, aptGetGroup},
			builderErr: true,
		},
		{
			event:              podExecEvent("p1", "c1", ""),
			groups:             []*storage.PolicyGroup{podExecGroup},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
			withProcessSection: true,
		},
		// PODS_ATTACH test cases
		{
			event:              podAttachEvent("p1", "c1"),
			groups:             []*storage.PolicyGroup{podAttachGroup},
			expectedViolations: []*storage.Alert_Violation{podAttachViolationMsg("p1", "c1")},
		},
		{
			event:              podAttachEvent("p1", ""),
			groups:             []*storage.PolicyGroup{podAttachGroup},
			expectedViolations: []*storage.Alert_Violation{podAttachViolationMsg("p1", "")},
		},
		{
			// No event provided, should not match
			groups: []*storage.PolicyGroup{podAttachGroup},
		},
		{
			// Port forward event should not match attach policy
			event:  podPortForwardEvent("p1", 8000),
			groups: []*storage.PolicyGroup{podAttachGroup},
		},
		{
			// Exec event should not match attach policy
			event:  podExecEvent("p1", "c1", "cmd"),
			groups: []*storage.PolicyGroup{podAttachGroup},
		},
		{
			// Attach event should not match exec policy
			event:  podAttachEvent("p1", "c1"),
			groups: []*storage.PolicyGroup{podExecGroup},
		},
		{
			// Attach policy with process group should fail builder
			event:      podAttachEvent("p1", "c1"),
			groups:     []*storage.PolicyGroup{podAttachGroup, aptGetGroup},
			builderErr: true,
		},
	} {
		suite.T().Run(fmt.Sprintf("%+v", c.groups), func(t *testing.T) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)
			if c.withProcessSection {
				policy.PolicySections = append(policy.PolicySections,
					&storage.PolicySection{PolicyGroups: []*storage.PolicyGroup{aptGetGroup}})
			}

			m, err := BuildKubeEventMatcher(policy)
			if c.builderErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			actualViolations, err := m.MatchKubeEvent(nil, c.event, &storage.Deployment{})
			suite.Require().NoError(err)

			assert.Nil(t, actualViolations.ProcessViolation)
			if len(c.expectedViolations) == 0 {
				assert.Nil(t, actualViolations.AlertViolations)
			} else {
				protoassert.ElementsMatch(t, c.expectedViolations, actualViolations.AlertViolations)
			}
		})
	}
}

func (suite *RuntimeCriteriaTestSuite) TestKubeEventDefaultPolicies() {
	for _, c := range []struct {
		policyName         string
		event              *storage.KubernetesEvent
		expectedViolations []*storage.Alert_Violation
	}{
		{
			policyName:         "Kubernetes Actions: Exec into Pod",
			event:              podExecEvent("p1", "c1", "apt-get"),
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "apt-get")},
		},
		{
			policyName: "Kubernetes Actions: Exec into Pod",
			event:      podPortForwardEvent("p1", 8000),
		},
		// Event without CREATE.
		{
			policyName: "Kubernetes Actions: Exec into Pod",
			event: &storage.KubernetesEvent{
				Object: &storage.KubernetesEvent_Object{
					Name:     "p1",
					Resource: storage.KubernetesEvent_Object_PODS_EXEC,
				},
				ObjectArgs: &storage.KubernetesEvent_PodExecArgs_{
					PodExecArgs: &storage.KubernetesEvent_PodExecArgs{
						Container: "c1",
					},
				},
			},
			expectedViolations: []*storage.Alert_Violation{podExecViolationMsg("p1", "c1", "")},
		},
		{
			policyName: "Kubernetes Actions: Port Forward to Pod",
		},
		{
			policyName:         "Kubernetes Actions: Port Forward to Pod",
			event:              podPortForwardEvent("p1", 8000),
			expectedViolations: []*storage.Alert_Violation{podPortForwardViolationMsg("p1", 8000)},
		},
		{
			policyName: "Kubernetes Actions: Port Forward to Pod",
			event: &storage.KubernetesEvent{
				Object: &storage.KubernetesEvent_Object{
					Name:     "p1",
					Resource: storage.KubernetesEvent_Object_PODS_PORTFORWARD,
				},
				ObjectArgs: &storage.KubernetesEvent_PodPortForwardArgs_{
					PodPortForwardArgs: &storage.KubernetesEvent_PodPortForwardArgs{
						Ports: []int32{8000},
					},
				},
			},
			expectedViolations: []*storage.Alert_Violation{podPortForwardViolationMsg("p1", 8000)},
		},
	} {
		suite.T().Run(fmt.Sprintf("%s:%s", c.policyName, kubernetes.EventAsString(c.event)), func(t *testing.T) {
			policy := suite.MustGetPolicy(c.policyName)
			m, err := BuildKubeEventMatcher(policy)
			require.NoError(t, err)

			actualViolations, err := m.MatchKubeEvent(nil, c.event, &storage.Deployment{})
			suite.Require().NoError(err)

			assert.Nil(t, actualViolations.ProcessViolation)
			if len(c.expectedViolations) == 0 {
				for _, a := range actualViolations.AlertViolations {
					fmt.Printf("%v", protoutils.NewWrapper(a))
				}

				assert.Nil(t, actualViolations.AlertViolations)
			} else {
				protoassert.ElementsMatch(t, c.expectedViolations, actualViolations.AlertViolations)
			}
		})
	}
}

func BenchmarkProcessPolicies(b *testing.B) {
	privilegedDep := fixtures.GetDeployment().CloneVT()
	privilegedDep.Id = "PRIVILEGED"
	images := []*storage.Image{fixtures.GetImage(), fixtures.GetImage()}

	nonPrivilegedDep := fixtures.GetDeployment().CloneVT()
	nonPrivilegedDep.Id = "NOTPRIVILEGED"
	nonPrivilegedDep.Containers[0].SecurityContext.Privileged = false

	const aptGetKey = "apt-get"
	const aptGet2Key = "apt-get2"
	const curlKey = "curl"
	const bashKey = "bash"

	indicators := make(map[string]map[string]*storage.ProcessIndicator)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		indicators[dep.GetId()] = map[string]*storage.ProcessIndicator{
			aptGetKey:  newIndicator(dep, "apt-get", "install nginx", "/bin/apt-get"),
			aptGet2Key: newIndicator(dep, "apt-get", "update", "/bin/apt-get"),
			curlKey:    newIndicator(dep, "curl", "https://stackrox.io", "/bin/curl"),
			bashKey:    newIndicator(dep, "bash", "attach.sh", "/bin/bash"),
		}
	}
	processesNotInBaseline := map[string]set.StringSet{
		privilegedDep.GetId():    set.NewStringSet(aptGetKey, aptGet2Key, bashKey),
		nonPrivilegedDep.GetId(): set.NewStringSet(aptGetKey, curlKey, bashKey),
	}

	// Plain groups
	aptGetGroup := policyGroupWithSingleKeyValue(fieldnames.ProcessName, "apt-get", false)
	privilegedGroup := policyGroupWithSingleKeyValue(fieldnames.PrivilegedContainer, "true", false)
	baselineGroup := policyGroupWithSingleKeyValue(fieldnames.UnexpectedProcessExecuted, "true", false)

	for _, testCase := range []struct {
		groups []*storage.PolicyGroup

		// Deployment ids to indicator keys
		expectedMatches        map[string][]string
		expectedProcessMatches map[string][]string
		// Deployment ids to violations
		expectedViolations map[string][]*storage.Alert_Violation
	}{
		{
			groups: []*storage.PolicyGroup{aptGetGroup},
			// only process violation, no alert violation
			expectedMatches: map[string][]string{},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key, bashKey},
				nonPrivilegedDep.GetId(): {aptGetKey, curlKey, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processBaselineMessage(privilegedDep, true, false, "apt-get", "apt-get", "bash"),
				nonPrivilegedDep.GetId(): processBaselineMessage(nonPrivilegedDep, true, false, "apt-get", "bash", "curl"),
			},
		},

		{
			groups: []*storage.PolicyGroup{privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, curlKey, bashKey},
			},
			expectedProcessMatches: map[string][]string{},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get", "curl", "bash"),
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId():    processBaselineMessage(privilegedDep, true, false, "apt-get", "apt-get"),
				nonPrivilegedDep.GetId(): processBaselineMessage(nonPrivilegedDep, true, false, "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId():    {aptGetKey, aptGet2Key},
				nonPrivilegedDep.GetId(): {aptGetKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, false, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
		{
			groups: []*storage.PolicyGroup{privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, true, true, "apt-get", "apt-get", "bash"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key, bashKey},
			},
		},
		{
			groups: []*storage.PolicyGroup{aptGetGroup, privilegedGroup, baselineGroup},
			expectedMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
			expectedViolations: map[string][]*storage.Alert_Violation{
				privilegedDep.GetId(): processBaselineMessage(privilegedDep, true, true, "apt-get", "apt-get"),
			},
			expectedProcessMatches: map[string][]string{
				privilegedDep.GetId(): {aptGetKey, aptGet2Key},
			},
		},
	} {
		c := testCase
		b.Run(fmt.Sprintf("%+v", c.groups), func(b *testing.B) {
			policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, c.groups...)
			m, err := BuildDeploymentWithProcessMatcher(policy)
			require.NoError(b, err)

			b.ResetTimer()
			for b.Loop() {
				for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
					for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
						_, err := m.MatchDeploymentWithProcess(nil, enhancedDeployment(dep, images), indicators[dep.GetId()][key], processesNotInBaseline[dep.GetId()].Contains(key))
						require.NoError(b, err)
					}
				}
			}
		})
	}

	policy := policyWithGroups(storage.EventSource_DEPLOYMENT_EVENT, aptGetGroup, privilegedGroup, baselineGroup)
	m, err := BuildDeploymentWithProcessMatcher(policy)
	require.NoError(b, err)
	for _, dep := range []*storage.Deployment{privilegedDep, nonPrivilegedDep} {
		for _, key := range []string{aptGetKey, aptGet2Key, curlKey, bashKey} {
			indicator := indicators[dep.GetId()][key]
			notInBaseline := processesNotInBaseline[dep.GetId()].Contains(key)
			b.Run(fmt.Sprintf("benchmark caching: %s/%s", dep.GetId(), key), func(b *testing.B) {
				var resNoCaching Violations
				b.Run("no caching", func(b *testing.B) {
					for b.Loop() {
						var err error
						resNoCaching, err = m.MatchDeploymentWithProcess(nil, enhancedDeployment(privilegedDep, images), indicator, notInBaseline)
						require.NoError(b, err)
					}
				})

				var resWithCaching Violations
				b.Run("with caching", func(b *testing.B) {
					var cache CacheReceptacle
					for b.Loop() {
						var err error
						resWithCaching, err = m.MatchDeploymentWithProcess(&cache, enhancedDeployment(privilegedDep, images), indicator, notInBaseline)
						require.NoError(b, err)
					}
				})
				assertViolations(b, resNoCaching, resWithCaching)
			})
		}
	}
}

func (suite *RuntimeCriteriaTestSuite) TestDeploymentFileAccess() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Deployment file open policy with matching event",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file open policy with mismatching event (UNLINK)",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file open policy with mismatching event (/tmp/foo)",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with negated file operation",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with multiple negated operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple files and single operation",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple files and multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with no operations",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd"),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with all allowed files",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with suffix",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/ssh/sshd_config-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/sudoers-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(suite.T())
		defer ResetFieldMetadataSingleton(suite.T())

		suite.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			suite.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDep := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDep, event.access)
				suite.Require().NoError(err)

				if event.expectAlert {
					suite.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					suite.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					suite.Require().NotNil(fileAccess, "expected file access info")

					suite.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					suite.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					suite.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					suite.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

func (suite *RuntimeCriteriaTestSuite) TestDeploymentEffectiveFileAccess() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Deployment effective file open policy with matching event",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file open policy with mismatching event (UNLINK)",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file open policy with mismatching event (/etc/sudoers)",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with negated file operation",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple negated operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple files and single operation",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple files and multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with no operations",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd"),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with all allowed files",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with suffix",
			policy:      newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT, nil, false, "/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      newEffectiveFileAccessEvent("/etc/passwd-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/shadow-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/ssh/sshd_config-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newEffectiveFileAccessEvent("/etc/sudoers-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(suite.T())
		defer ResetFieldMetadataSingleton(suite.T())

		suite.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			suite.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDep := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDep, event.access)
				suite.Require().NoError(err)

				if event.expectAlert {
					suite.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					suite.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					suite.Require().NotNil(fileAccess, "expected file access info")

					suite.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					suite.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					suite.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					suite.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

func (suite *RuntimeCriteriaTestSuite) TestDeploymentDualPathMatching() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Event with both paths - policy matches actual path only",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy matches effective only",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH paths (AND within section)",
			policy:      newDualPathPolicy("/etc/passwd", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section policy - first section matches (OR behavior)",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section policy - second section matches (OR behavior)",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with mixed path types - actual path section matches",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with mixed path types - effective path section matches",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with dual paths in one section - complex AND/OR",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values: []*storage.PolicyValue{
								{Value: "/etc/passwd"},
								{Value: "/etc/shadow"},
							},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/ssh/sshd_config"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					// Matches section 1 (both actual and effective paths match)
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},

		// Invalid/edge cases - unexpected behaviors
		{
			description: "Event with both paths - policy matches neither",
			policy: newFileAccessPolicy(storage.EventSource_DEPLOYMENT_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/sudoers",
			),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Event with both paths - policy requires EITHER and only actual path matches",
			policy:      newDualPathPolicy("/etc/passwd", "/etc/sudoers", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy requires EITHER and only effective matches",
			policy:      newDualPathPolicy("/etc/sudoers", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy requires EITHER and only BOTH match",
			policy:      newDualPathPolicy("/etc/sudoers", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/sudoers", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH but operation doesn't match",
			policy:      newDualPathPolicy("/etc/passwd", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_CREATE}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Multi-section policy - no sections match",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/ssh/sshd_config"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Multi-section with dual paths - neither section matches completely",
			policy: newMultiSectionPolicy(storage.EventSource_DEPLOYMENT_EVENT, []*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}, {Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "UNLINK"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.FilePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}, {Value: "/etc/ssh/sshd_config"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "UNLINK"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					// Section 1: actual matches, effective doesn't (AND fails)
					// Section 2: actual doesn't match, effective does (AND fails)
					// Overall: no section fully matches (OR fails)
					access:      newDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(suite.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(suite.T())
		defer ResetFieldMetadataSingleton(suite.T())

		suite.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			suite.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDep := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDep, event.access)
				suite.Require().NoError(err)

				if event.expectAlert {
					suite.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					suite.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					suite.Require().NotNil(fileAccess, "expected file access info")

					suite.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					suite.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					suite.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					suite.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

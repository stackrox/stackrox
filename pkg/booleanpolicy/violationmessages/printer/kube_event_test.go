package printer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestViolationMessageForAuditLogEvents(t *testing.T) {
	cases := []struct {
		testName    string
		res         storage.KubernetesEvent_Object_Resource
		verb        storage.KubernetesEvent_APIVerb
		name        string
		expectedMsg string
	}{
		{"Valid message when event is GET on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_GET, "hush hush", "Access to secret \"hush hush\" in namespace \"ns\""},
		{"Valid message when event is CREATE on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_CREATE, "hush hush", "Access to secret \"hush hush\" in namespace \"ns\""},
		{"Valid message when event is LIST on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_LIST, "", "Access to secrets in namespace \"ns\""},
		{"Valid message when event is WATCH on a specific secret", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_WATCH, "hush hush", "Access to secret \"hush hush\" in namespace \"ns\""},
		{"Valid message when event is WATCH on all secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_WATCH, "", "Access to secrets in namespace \"ns\""},
		{"Valid message when event is GET on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_GET, "all the configs", "Access to configmap \"all the configs\" in namespace \"ns\""},
		{"Valid message when event is DELETE on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_DELETE, "all the configs", "Access to configmap \"all the configs\" in namespace \"ns\""},
		{"Valid message when event is LIST on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_LIST, "", "Access to configmaps in namespace \"ns\""},
		{"Valid message when event is WATCH on a specific configmap", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_WATCH, "all the configs", "Access to configmap \"all the configs\" in namespace \"ns\""},
		{"Valid message when event is WATCH on all configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_WATCH, "", "Access to configmaps in namespace \"ns\""},
		{"Valid message when event is CREATE on clusterrole", storage.KubernetesEvent_Object_CLUSTER_ROLES, storage.KubernetesEvent_CREATE, "test cr", "Access to cluster role \"test cr\" in namespace \"ns\""},
		{"Valid message when event is DELETE on netpol", storage.KubernetesEvent_Object_NETWORK_POLICIES, storage.KubernetesEvent_DELETE, "test netpol", "Access to network policy \"test netpol\" in namespace \"ns\""},
		{"Valid message when event is CREATE on a network policy", storage.KubernetesEvent_Object_NETWORK_POLICIES, storage.KubernetesEvent_CREATE, "new-net-pol", "Access to network policy \"new-net-pol\" in namespace \"ns\""},
		{"Valid message when event is LIST on all network policies", storage.KubernetesEvent_Object_NETWORK_POLICIES, storage.KubernetesEvent_LIST, "", "Access to network policies in namespace \"ns\""},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			kubeEvent := getKubeEvent(c.res, c.verb, "cluster-id", "ns", c.name)
			violation := GenerateKubeEventViolationMsg(kubeEvent)
			assert.Equal(t, c.expectedMsg, violation.GetMessage())
		})
	}
}

func TestViolationAttrsForAuditLogEvents(t *testing.T) {
	expectedAttrs := map[string]string{
		APIVerbKey:     "LIST",
		UsernameKey:    "username",
		UserGroupsKey:  "groupA, groupB",
		ResourceURIKey: "/api/v1/namespaces/namespace/secrets",
		UserAgentKey:   "curl",
		IPAddressKey:   "192.168.1.1, 127.0.0.1",
	}

	kubeEvent := getKubeEvent(storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_LIST, "cluster-id", "namespace", "the-secret")
	validateViolationAttrs(t, kubeEvent, expectedAttrs)
}

func TestViolationAttrsForAuditLogEventsAddsImpersonatedUser(t *testing.T) {
	expectedAttrs := map[string]string{
		APIVerbKey:                "GET",
		UsernameKey:               "username",
		UserGroupsKey:             "groupA, groupB",
		ResourceURIKey:            "/api/v1/namespaces/namespace/secrets/the-secret",
		UserAgentKey:              "curl",
		IPAddressKey:              "192.168.1.1, 127.0.0.1",
		ImpersonatedUsernameKey:   "test-service-account",
		ImpersonatedUserGroupsKey: "service-accounts, groupC",
	}

	kubeEvent := getKubeEvent(storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_GET, "cluster-id", "namespace", "the-secret")
	kubeEvent.ImpersonatedUser = &storage.KubernetesEvent_User{
		Username: "test-service-account",
		Groups:   []string{"service-accounts", "groupC"},
	}
	validateViolationAttrs(t, kubeEvent, expectedAttrs)
}

func validateViolationAttrs(t *testing.T, kubeEvent *storage.KubernetesEvent, expectedAttrs map[string]string) {
	violation := GenerateKubeEventViolationMsg(kubeEvent)
	attrs := violation.GetKeyValueAttrs().GetAttrs()

	// Should have exactly as many elements as expected
	assert.Len(t, attrs, len(expectedAttrs))

	for _, a := range attrs {
		assert.Equal(t, expectedAttrs[a.GetKey()], a.GetValue())
	}
}

func TestViolationMessageForPodExecEvents(t *testing.T) {
	cases := []struct {
		testName    string
		podName     string
		container   string
		commands    []string
		expectedMsg string
	}{
		{
			"Exec with pod, container, and commands",
			"my-pod",
			"my-container",
			[]string{"ls", "-l"},
			"Kubernetes API received exec 'ls -l' request into pod 'my-pod' container 'my-container'",
		},
		{
			"Exec with pod and commands, no container",
			"my-pod",
			"",
			[]string{"cat", "/etc/passwd"},
			"Kubernetes API received exec 'cat /etc/passwd' request into pod 'my-pod'",
		},
		{
			"Exec with pod and container, no commands",
			"my-pod",
			"my-container",
			nil,
			"Kubernetes API received exec request into pod 'my-pod' container 'my-container'",
		},
		{
			"Exec with pod only",
			"my-pod",
			"",
			nil,
			"Kubernetes API received exec request into pod 'my-pod'",
		},
		{
			"Exec with no details",
			"",
			"",
			nil,
			"Kubernetes API received exec request",
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			kubeEvent := getPodExecEvent(c.podName, c.container, c.commands)
			violation := GenerateKubeEventViolationMsg(kubeEvent)
			assert.Equal(t, c.expectedMsg, violation.GetMessage())
			assert.Equal(t, storage.Alert_Violation_K8S_EVENT, violation.GetType())
		})
	}
}

func TestViolationAttrsForPodExecEvents(t *testing.T) {
	kubeEvent := getPodExecEvent("my-pod", "my-container", []string{"ls", "-l"})
	violation := GenerateKubeEventViolationMsg(kubeEvent)
	attrs := violation.GetKeyValueAttrs().GetAttrs()

	expectedAttrs := map[string]string{
		PodKey:        "my-pod",
		ContainerKey:  "my-container",
		CommandsKey:   "ls -l",
		UsernameKey:   "username",
		UserGroupsKey: "groupA, groupB",
		UserAgentKey:  "curl",
		IPAddressKey:  "192.168.1.1, 127.0.0.1",
	}

	assert.Len(t, attrs, len(expectedAttrs))
	for _, a := range attrs {
		assert.Equal(t, expectedAttrs[a.GetKey()], a.GetValue())
	}
}

func TestViolationMessageForPodPortForwardEvents(t *testing.T) {
	cases := []struct {
		testName    string
		podName     string
		ports       []int32
		expectedMsg string
	}{
		{
			"Port forward with pod and ports",
			"my-pod",
			[]int32{8080, 9090},
			"Kubernetes API received port forward request to pod 'my-pod' ports '8080, 9090'",
		},
		{
			"Port forward with pod and single port",
			"my-pod",
			[]int32{8080},
			"Kubernetes API received port forward request to pod 'my-pod' ports '8080'",
		},
		{
			"Port forward with pod only",
			"my-pod",
			nil,
			"Kubernetes API received port forward request to pod 'my-pod'",
		},
		{
			"Port forward with no details",
			"",
			nil,
			"Kubernetes API received port forward request",
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			kubeEvent := getPodPortForwardEvent(c.podName, c.ports)
			violation := GenerateKubeEventViolationMsg(kubeEvent)
			assert.Equal(t, c.expectedMsg, violation.GetMessage())
			assert.Equal(t, storage.Alert_Violation_K8S_EVENT, violation.GetType())
		})
	}
}

func TestViolationAttrsForPodPortForwardEvents(t *testing.T) {
	kubeEvent := getPodPortForwardEvent("my-pod", []int32{8080, 9090})
	violation := GenerateKubeEventViolationMsg(kubeEvent)
	attrs := violation.GetKeyValueAttrs().GetAttrs()

	expectedAttrs := map[string]string{
		PodKey:        "my-pod",
		PortsKey:      "8080, 9090",
		UsernameKey:   "username",
		UserGroupsKey: "groupA, groupB",
		UserAgentKey:  "curl",
		IPAddressKey:  "192.168.1.1, 127.0.0.1",
	}

	assert.Len(t, attrs, len(expectedAttrs))
	for _, a := range attrs {
		assert.Equal(t, expectedAttrs[a.GetKey()], a.GetValue())
	}
}

func TestViolationMessageForPodAttachEvents(t *testing.T) {
	cases := []struct {
		testName    string
		podName     string
		container   string
		expectedMsg string
	}{
		{
			"Attach with pod and container",
			"my-pod",
			"my-container",
			"Kubernetes API received attach request to pod 'my-pod' container 'my-container'",
		},
		{
			"Attach with pod only",
			"my-pod",
			"",
			"Kubernetes API received attach request to pod 'my-pod'",
		},
		{
			"Attach with no details",
			"",
			"",
			"Kubernetes API received attach request",
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			kubeEvent := getPodAttachEvent(c.podName, c.container)
			violation := GenerateKubeEventViolationMsg(kubeEvent)
			assert.Equal(t, c.expectedMsg, violation.GetMessage())
			assert.Equal(t, storage.Alert_Violation_K8S_EVENT, violation.GetType())
		})
	}
}

func TestViolationAttrsForPodAttachEvents(t *testing.T) {
	kubeEvent := getPodAttachEvent("my-pod", "my-container")
	violation := GenerateKubeEventViolationMsg(kubeEvent)
	attrs := violation.GetKeyValueAttrs().GetAttrs()

	expectedAttrs := map[string]string{
		PodKey:        "my-pod",
		ContainerKey:  "my-container",
		UsernameKey:   "username",
		UserGroupsKey: "groupA, groupB",
		UserAgentKey:  "curl",
		IPAddressKey:  "192.168.1.1, 127.0.0.1",
	}

	assert.Len(t, attrs, len(expectedAttrs))
	for _, a := range attrs {
		assert.Equal(t, expectedAttrs[a.GetKey()], a.GetValue())
	}
}

func getKubeEvent(resource storage.KubernetesEvent_Object_Resource, verb storage.KubernetesEvent_APIVerb, clusterID, namespace, name string) *storage.KubernetesEvent {
	requestURI := fmt.Sprintf("/api/v1/namespaces/%s/%s/%s", namespace, strings.ToLower(resource.String()), name)
	if verb == storage.KubernetesEvent_LIST {
		requestURI = fmt.Sprintf("/api/v1/namespaces/%s/%s?limit=500", namespace, strings.ToLower(resource.String()))
	}
	return &storage.KubernetesEvent{
		Id: uuid.NewV4().String(),
		Object: &storage.KubernetesEvent_Object{
			Name:      name,
			Resource:  resource,
			ClusterId: clusterID,
			Namespace: namespace,
		},
		Timestamp: protocompat.TimestampNow(),
		ApiVerb:   verb,
		User: &storage.KubernetesEvent_User{
			Username: "username",
			Groups:   []string{"groupA", "groupB"},
		},
		SourceIps: []string{"192.168.1.1", "127.0.0.1"},
		UserAgent: "curl",
		ResponseStatus: &storage.KubernetesEvent_ResponseStatus{
			StatusCode: 200,
			Reason:     "cause",
		},
		RequestUri: requestURI,
	}
}

func getBaseKubeEvent() *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Id:        uuid.NewV4().String(),
		Timestamp: protocompat.TimestampNow(),
		User: &storage.KubernetesEvent_User{
			Username: "username",
			Groups:   []string{"groupA", "groupB"},
		},
		SourceIps: []string{"192.168.1.1", "127.0.0.1"},
		UserAgent: "curl",
		ResponseStatus: &storage.KubernetesEvent_ResponseStatus{
			StatusCode: 200,
			Reason:     "OK",
		},
	}
}

func getPodExecEvent(podName, container string, commands []string) *storage.KubernetesEvent {
	event := getBaseKubeEvent()
	event.Object = &storage.KubernetesEvent_Object{
		Name:      podName,
		Resource:  storage.KubernetesEvent_Object_PODS_EXEC,
		ClusterId: "cluster-id",
		Namespace: "ns",
	}
	event.ApiVerb = storage.KubernetesEvent_CREATE
	event.ObjectArgs = &storage.KubernetesEvent_PodExecArgs_{
		PodExecArgs: &storage.KubernetesEvent_PodExecArgs{
			Container: container,
			Commands:  commands,
		},
	}
	return event
}

func getPodPortForwardEvent(podName string, ports []int32) *storage.KubernetesEvent {
	event := getBaseKubeEvent()
	event.Object = &storage.KubernetesEvent_Object{
		Name:      podName,
		Resource:  storage.KubernetesEvent_Object_PODS_PORTFORWARD,
		ClusterId: "cluster-id",
		Namespace: "ns",
	}
	event.ApiVerb = storage.KubernetesEvent_CREATE
	event.ObjectArgs = &storage.KubernetesEvent_PodPortForwardArgs_{
		PodPortForwardArgs: &storage.KubernetesEvent_PodPortForwardArgs{
			Ports: ports,
		},
	}
	return event
}

func getPodAttachEvent(podName, container string) *storage.KubernetesEvent {
	event := getBaseKubeEvent()
	event.Object = &storage.KubernetesEvent_Object{
		Name:      podName,
		Resource:  storage.KubernetesEvent_Object_PODS_ATTACH,
		ClusterId: "cluster-id",
		Namespace: "ns",
	}
	event.ApiVerb = storage.KubernetesEvent_CREATE
	event.ObjectArgs = &storage.KubernetesEvent_PodAttachArgs_{
		PodAttachArgs: &storage.KubernetesEvent_PodAttachArgs{
			Container: container,
		},
	}
	return event
}

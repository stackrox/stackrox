package printer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
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
		{"Valid message when event is GET on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_GET, "hush hush", "Access to secret \"hush hush\" in \"namespace\""},
		{"Valid message when event is CREATE on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_CREATE, "hush hush", "Access to secret \"hush hush\" in \"namespace\""},
		{"Valid message when event is LIST on secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_LIST, "", "Access to secrets in \"namespace\""},
		{"Valid message when event is WATCH on a specific secret", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_WATCH, "hush hush", "Access to secret \"hush hush\" in \"namespace\""},
		{"Valid message when event is WATCH on all secrets", storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_WATCH, "", "Access to secrets in \"namespace\""},
		{"Valid message when event is GET on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_GET, "all the configs", "Access to configmap \"all the configs\" in \"namespace\""},
		{"Valid message when event is DELETE on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_DELETE, "all the configs", "Access to configmap \"all the configs\" in \"namespace\""},
		{"Valid message when event is LIST on configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_LIST, "", "Access to configmaps in \"namespace\""},
		{"Valid message when event is WATCH on a specific configmap", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_WATCH, "all the configs", "Access to configmap \"all the configs\" in \"namespace\""},
		{"Valid message when event is WATCH on all configmaps", storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_WATCH, "", "Access to configmaps in \"namespace\""},
		{"Valid message when event is CREATE on clusterrole", storage.KubernetesEvent_Object_CLUSTER_ROLES, storage.KubernetesEvent_CREATE, "test cr", "Access to clusterrole \"test cr\" in \"namespace\""},
		{"Valid message when event is DELETE on netpol", storage.KubernetesEvent_Object_NETWORK_POLICIES, storage.KubernetesEvent_DELETE, "test netpol", "Access to networkpolicy \"test netpol\" in \"namespace\""},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			kubeEvent := getKubeEvent(c.res, c.verb, "cluster-id", "namespace", c.name)
			violation := GenerateKubeEventViolationMsg(kubeEvent)
			assert.Equal(t, c.expectedMsg, violation.Message)
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
		Timestamp: types.TimestampNow(),
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

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
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername("test-service-account")
	ku.SetGroups([]string{"service-accounts", "groupC"})
	kubeEvent.SetImpersonatedUser(ku)
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
	ko := &storage.KubernetesEvent_Object{}
	ko.SetName(name)
	ko.SetResource(resource)
	ko.SetClusterId(clusterID)
	ko.SetNamespace(namespace)
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername("username")
	ku.SetGroups([]string{"groupA", "groupB"})
	kr := &storage.KubernetesEvent_ResponseStatus{}
	kr.SetStatusCode(200)
	kr.SetReason("cause")
	ke := &storage.KubernetesEvent{}
	ke.SetId(uuid.NewV4().String())
	ke.SetObject(ko)
	ke.SetTimestamp(protocompat.TimestampNow())
	ke.SetApiVerb(verb)
	ke.SetUser(ku)
	ke.SetSourceIps([]string{"192.168.1.1", "127.0.0.1"})
	ke.SetUserAgent("curl")
	ke.SetResponseStatus(kr)
	ke.SetRequestUri(requestURI)
	return ke
}

package plan

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	serviceAccountFromBundleYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackrox
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	modifiedServiceAccountFromBundleYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: stackroxxx
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
imagePullSecrets:
- name: stackrox
`

	liveServiceAccountYAML = `
apiVersion: v1
imagePullSecrets:
- name: stackrox
kind: ServiceAccount
metadata:
  creationTimestamp: "2019-08-29T08:45:47Z"
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: sensor
  annotations:
    sensor-upgrader.stackrox.io/last-upgrade-id: abcd
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
        f:labels:
          .: {}
          f:app.kubernetes.io/name: {}
          f:auto-upgrade.stackrox.io/component: {}
      f:spec:
        f:ports:
          .: {}
          k:{"port":443,"protocol":"TCP"}:
            .: {}
            f:name: {}
            f:port: {}
            f:protocol: {}
            f:targetPort: {}
        f:selector:
          .: {}
          f:app: {}
        f:sessionAffinity: {}
        f:type: {}
    manager: kubectl
    operation: Update
    time: "2020-08-07T20:45:25Z"
  name: sensor
  namespace: stackrox
  resourceVersion: "444536"
  selfLink: /api/v1/namespaces/stackrox/serviceaccounts/sensor
  uid: 6543d0c6-ca39-11e9-a14d-025000000001
secrets:
- name: sensor-token-dsvb7
`

	admissionWebhookYAMLBase = `apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: stackrox
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
webhooks:
  - name: policyeval.stackrox.io
    rules:
      - apiGroups:
          - '*'
        apiVersions:
          - '*'
        operations:
          - CREATE
        resources:
          - pods
          - deployments
          - replicasets
          - replicationcontrollers
          - statefulsets
          - daemonsets
        scope: '*'
    namespaceSelector:
       matchExpressions:
       - key: namespace.metadata.stackrox.io/name
         operator: NotIn
         values:
         - stackrox
         - kube-system
         - kube-public
    failurePolicy: Ignore
    clientConfig:
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUIwVENDQVhhZ0F3SUJBZ0lVSHhpVnlDalFUc1Ixci9xRmFRV0JqS1pVKzlBd0NnWUlLb1pJemowRUF3SXcKUmpFbk1DVUdBMVVFQXhNZVUzUmhZMnRTYjNnZ1EyVnlkR2xtYVdOaGRHVWdRWFYwYUc5eWFYUjVNUnN3R1FZRApWUVFGRXhJNE56VTVPVFUwTmprek9EVTNNRGN4T0RBd0hoY05NVGt3T1RFeU1UZ3hOREF3V2hjTk1qUXdPVEV3Ck1UZ3hOREF3V2pCR01TY3dKUVlEVlFRREV4NVRkR0ZqYTFKdmVDQkRaWEowYVdacFkyRjBaU0JCZFhSb2IzSnAKZEhreEd6QVpCZ05WQkFVVEVqZzNOVGs1TlRRMk9UTTROVGN3TnpFNE1EQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxRwpTTTQ5QXdFSEEwSUFCTWxoSjhHQXR2dVZRSHE2bUpzTHE3T0VqT2grNDJMcHQrVlBvdG1qKzN2cWJRUmc0eHhyCk03YzFUUzltTi9GMktEbzdsTUEzR1JKYldib2s4SVFyMzVhalFqQkFNQTRHQTFVZER3RUIvd1FFQXdJQkJqQVAKQmdOVkhSTUJBZjhFQlRBREFRSC9NQjBHQTFVZERnUVdCQlFvRnJLSStLY1JMQm4yUENyWnNYNUhnbmJzYkRBSwpCZ2dxaGtqT1BRUURBZ05KQURCR0FpRUFsYUFvcnZBNHlVVkh0M0diNFFqSDhyQklIbU5aSXd1bXBFWDA2ZkZ6CmVmVUNJUUNWY244dFFISmJkZDNSL3dOaXA4dSs3bEs2R0p3TldGc0FFdytBeXE4QU1RPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
      service:
        namespace: stackrox
        name: sensor-webhook
        path: /admissioncontroller`

	admissionWebhookYAMLWithDefaults = admissionWebhookYAMLBase + `
    admissionReviewVersions:
      - "v1beta1"
    timeoutSeconds: 30
    sideEffects: "Unknown"
`
	admissionWebhookYAMLWithNonDefaultAdmissionReviewVersions = admissionWebhookYAMLBase + `
    admissionReviewVersions:
      - "v1beta1"
      - "v1beta2"
    timeoutSeconds: 30
    sideEffects: "Unknown"
`

	admissionWebhookYAMLWithNonDefaultAdmissionReviewVersions2 = admissionWebhookYAMLBase + `
    admissionReviewVersions:
      - "v1beta2"
    timeoutSeconds: 30
    sideEffects: "Unknown"
`

	admissionWebhookYAMLWithNonDefaultTimeout = admissionWebhookYAMLBase + `
    admissionReviewVersions:
      - "v1beta1"
    timeoutSeconds: 60
    sideEffects: "Unknown"
`

	admissionWebhookYAMLWithNonDefaultSideEffects = admissionWebhookYAMLBase + `
    admissionReviewVersions:
      - "v1beta1"
    timeoutSeconds: 30
    sideEffects: "None"
`

	admissionWebhookYAMLWithDifferentRuleScopes = `apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: stackrox
  labels:
    app.kubernetes.io/name: stackrox
    auto-upgrade.stackrox.io/component: "sensor"
webhooks:
  - name: policyeval.stackrox.io
    rules:
      - apiGroups:
          - '*'
        apiVersions:
          - '*'
        operations:
          - CREATE
        resources:
          - pods
          - deployments
          - replicasets
          - replicationcontrollers
          - statefulsets
          - daemonsets
        scope: 'Cluster'
      - apiGroups:
          - '*'
        apiVersions:
          - '*'
        operations:
          - CREATE
        resources:
          - pods
          - deployments
          - replicasets
          - replicationcontrollers
          - statefulsets
          - daemonsets
        scope: '*'
      - apiGroups:
          - '*'
        apiVersions:
          - '*'
        operations:
          - CREATE
        resources:
          - pods
          - deployments
          - replicasets
          - replicationcontrollers
          - statefulsets
          - daemonsets
        scope: 'Namespaced'
    namespaceSelector:
       matchExpressions:
       - key: namespace.metadata.stackrox.io/name
         operator: NotIn
         values:
         - stackrox
         - kube-system
         - kube-public
    failurePolicy: Ignore
    clientConfig:
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUIwVENDQVhhZ0F3SUJBZ0lVSHhpVnlDalFUc1Ixci9xRmFRV0JqS1pVKzlBd0NnWUlLb1pJemowRUF3SXcKUmpFbk1DVUdBMVVFQXhNZVUzUmhZMnRTYjNnZ1EyVnlkR2xtYVdOaGRHVWdRWFYwYUc5eWFYUjVNUnN3R1FZRApWUVFGRXhJNE56VTVPVFUwTmprek9EVTNNRGN4T0RBd0hoY05NVGt3T1RFeU1UZ3hOREF3V2hjTk1qUXdPVEV3Ck1UZ3hOREF3V2pCR01TY3dKUVlEVlFRREV4NVRkR0ZqYTFKdmVDQkRaWEowYVdacFkyRjBaU0JCZFhSb2IzSnAKZEhreEd6QVpCZ05WQkFVVEVqZzNOVGs1TlRRMk9UTTROVGN3TnpFNE1EQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxRwpTTTQ5QXdFSEEwSUFCTWxoSjhHQXR2dVZRSHE2bUpzTHE3T0VqT2grNDJMcHQrVlBvdG1qKzN2cWJRUmc0eHhyCk03YzFUUzltTi9GMktEbzdsTUEzR1JKYldib2s4SVFyMzVhalFqQkFNQTRHQTFVZER3RUIvd1FFQXdJQkJqQVAKQmdOVkhSTUJBZjhFQlRBREFRSC9NQjBHQTFVZERnUVdCQlFvRnJLSStLY1JMQm4yUENyWnNYNUhnbmJzYkRBSwpCZ2dxaGtqT1BRUURBZ05KQURCR0FpRUFsYUFvcnZBNHlVVkh0M0diNFFqSDhyQklIbU5aSXd1bXBFWDA2ZkZ6CmVmVUNJUUNWY244dFFISmJkZDNSL3dOaXA4dSs3bEs2R0p3TldGc0FFdytBeXE4QU1RPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
      service:
        namespace: stackrox
        name: sensor-webhook
        path: /admissioncontroller`
)

func fromYAML(t *testing.T, yamlStr string) *unstructured.Unstructured {
	obj, err := k8sutil.UnstructuredFromYAML(yamlStr)
	require.NoError(t, err)
	return obj
}

func TestNormalizeObjects_EqualAfterNormalize(t *testing.T) {
	t.Parallel()

	liveSA := fromYAML(t, liveServiceAccountYAML)
	saFromBundle := fromYAML(t, serviceAccountFromBundleYAML)

	normalizeObject(liveSA)
	normalizeObject(saFromBundle)
	assert.Equal(t, liveSA, saFromBundle)
}

func TestNormalizeObjects_NotEqualAfterNormalize(t *testing.T) {
	t.Parallel()

	liveSA := fromYAML(t, liveServiceAccountYAML)
	modifiedSAFromBundle := fromYAML(t, modifiedServiceAccountFromBundleYAML)

	normalizeObject(liveSA)
	normalizeObject(modifiedSAFromBundle)
	assert.NotEqual(t, liveSA, modifiedSAFromBundle)
}

func TestNormalizeAdmissionController(t *testing.T) {
	t.Parallel()

	baseKeys := []string{"clientConfig", "failurePolicy", "name", "namespaceSelector", "rules"}

	testCases := []struct {
		yaml         string
		expectedKeys []string
	}{
		{
			admissionWebhookYAMLBase,
			baseKeys,
		},
		{
			admissionWebhookYAMLWithDefaults,
			baseKeys,
		},
		{
			admissionWebhookYAMLWithNonDefaultAdmissionReviewVersions,
			append([]string{"admissionReviewVersions"}, baseKeys...),
		},
		{
			admissionWebhookYAMLWithNonDefaultAdmissionReviewVersions2,
			append([]string{"admissionReviewVersions"}, baseKeys...),
		},
		{
			admissionWebhookYAMLWithNonDefaultTimeout,
			append([]string{"timeoutSeconds"}, baseKeys...),
		},
		{
			admissionWebhookYAMLWithNonDefaultSideEffects,
			append([]string{"sideEffects"}, baseKeys...),
		},
	}

	for i, testCase := range testCases {
		c := testCase
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			webhook := fromYAML(t, c.yaml)
			normalizeObject(webhook)
			webhooks := webhook.Object["webhooks"].([]interface{})
			assert.Len(t, webhooks, 1)
			firstWebhook := webhooks[0].(map[string]interface{})
			keys := make([]string, 0, len(firstWebhook))
			for k := range firstWebhook {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, c.expectedKeys, keys, "%+v != %+v", c.expectedKeys, keys)

			assert.NotContains(t, firstWebhook["rules"].([]interface{})[0].(map[string]interface{}), "scope")
		})
	}

	webhook := fromYAML(t, admissionWebhookYAMLWithDifferentRuleScopes)
	normalizeObject(webhook)
	rules := webhook.Object["webhooks"].([]interface{})[0].(map[string]interface{})["rules"].([]interface{})
	expectedScopes := []string{"Cluster", "*", "Namespaced"}
	for i, r := range rules {
		scope := r.(map[string]interface{})["scope"]
		if expectedScopes[i] == "*" {
			assert.Nil(t, scope)
		} else {
			assert.Equal(t, expectedScopes[i], scope.(string))
		}
	}
}

package plan

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	serviceAccountGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "ServiceAccount",
	}
	serviceGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "Service",
	}

	validatingAdmissionWebhookGVK = schema.GroupVersionKind{
		Group:   "admissionregistration.k8s.io",
		Version: "v1beta1",
		Kind:    "ValidatingWebhookConfiguration",
	}
)

func clearServiceAccountDynamicFields(obj *unstructured.Unstructured) {
	// Remove the default token, as it is auto-populated.
	defaultTokenNamePrefix := fmt.Sprintf("%s-token-", obj.GetName())
	secrets, _, _ := unstructured.NestedSlice(obj.Object, "secrets")
	var filteredSecrets []interface{}
	for _, secret := range secrets {
		secretObj, _ := secret.(map[string]interface{})
		if secretName, _, _ := unstructured.NestedString(secretObj, "name"); !strings.HasPrefix(secretName, defaultTokenNamePrefix) {
			filteredSecrets = append(filteredSecrets, secret)
		}
	}
	utils.Should(unstructured.SetNestedSlice(obj.Object, filteredSecrets, "secrets"))
}

func deleteValueIfMatching(m map[string]interface{}, key string, defaultValue interface{}) {
	if m[key] == defaultValue {
		delete(m, key)
	}
}

// The admission controller has a beta API, and our client code does some weird things where it removes values
// that match the defaults on YAMLs but not on objects retrieved from the API.
// Separately, the upgrader is currently unable to handle admission controller changes, and we want to make it
// avoid having to update admission controllers if possible. To facilitate this, we zero out values of specific
// fields in the admission controller if they match the API defaults.
func clearAdmissionWebhookDefaultFields(obj *unstructured.Unstructured) {
	webhooks, found, err := unstructured.NestedSlice(obj.Object, "webhooks")
	if err != nil {
		log.Errorf("Couldn't get webhooks field in validating admission webhook configuration: %v", err)
		return
	}
	if !found {
		log.Error("No webhooks field found in validating admission webhook configuration")
		return
	}
	for _, webhookInterface := range webhooks {
		webhook, ok := webhookInterface.(map[string]interface{})
		if !ok {
			log.Errorf("Webhook %+v was not a map[string]interface{}", webhook)
			return
		}
		deleteValueIfMatching(webhook, "timeoutSeconds", int64(30))
		deleteValueIfMatching(webhook, "sideEffects", "Unknown")
		admissionReviewVersions, ok := webhook["admissionReviewVersions"].([]interface{})
		if ok {
			if len(admissionReviewVersions) == 1 && admissionReviewVersions[0] == "v1beta1" {
				delete(webhook, "admissionReviewVersions")
			}
		}

		rules, ok := webhook["rules"].([]interface{})
		if ok {
			for _, r := range rules {
				typedRule, ok := r.(map[string]interface{})
				if !ok {
					log.Errorf("Rule in webhook %+v was not a map[string]interface{}", webhook)
					return
				}
				deleteValueIfMatching(typedRule, "scope", "*")
			}
		}
	}
	err = unstructured.SetNestedSlice(obj.Object, webhooks, "webhooks")
	if err != nil {
		log.Errorf("Failed to set webhooks field in validating admission webhook configuration: %v", err)
		return
	}
}

var (
	clearDynamicFieldsByGVK = map[schema.GroupVersionKind]func(*unstructured.Unstructured){
		serviceAccountGVK:             clearServiceAccountDynamicFields,
		validatingAdmissionWebhookGVK: clearAdmissionWebhookDefaultFields,
	}
)

// normalizeObject heuristically clears dynamic fields to increase the likelihood of objects being recognized as equal.
func normalizeObject(obj *unstructured.Unstructured) {
	obj.SetUID("")
	obj.SetSelfLink("")
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	k8sutil.DeleteAnnotation(obj, common.LastUpgradeIDAnnotationKey)
	unstructured.RemoveNestedField(obj.Object, "status")
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")

	if clearDynamicFieldsFn := clearDynamicFieldsByGVK[obj.GetObjectKind().GroupVersionKind()]; clearDynamicFieldsFn != nil {
		clearDynamicFieldsFn(obj)
	}
}

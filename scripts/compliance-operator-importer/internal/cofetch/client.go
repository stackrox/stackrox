package cofetch

import (
	"context"
	"errors"
	"fmt"

	"github.com/stackrox/co-acs-importer/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// GVRs for Compliance Operator resources.
var (
	scanSettingBindingGVR = schema.GroupVersionResource{
		Group:    "compliance.openshift.io",
		Version:  "v1alpha1",
		Resource: "scansettingbindings",
	}
	scanSettingGVR = schema.GroupVersionResource{
		Group:    "compliance.openshift.io",
		Version:  "v1alpha1",
		Resource: "scansettings",
	}
)

// k8sClient is the production implementation of COClient backed by a dynamic k8s client.
type k8sClient struct {
	dynamic   dynamic.Interface
	namespace string // empty string means all namespaces
}

// NewClient creates a COClient using the kube context specified in cfg.
// If cfg.KubeContext is empty the current context is used.
// If cfg.COAllNamespaces is true, resources are listed across all namespaces.
func NewClient(cfg *models.Config) (COClient, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if cfg.KubeContext != "" {
		overrides.CurrentContext = cfg.KubeContext
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	ns := cfg.CONamespace
	if cfg.COAllNamespaces {
		ns = ""
	}

	return &k8sClient{
		dynamic:   dynClient,
		namespace: ns,
	}, nil
}

// ListScanSettingBindings returns all ScanSettingBindings from the configured namespace(s).
func (c *k8sClient) ListScanSettingBindings(ctx context.Context) ([]ScanSettingBinding, error) {
	list, err := c.dynamic.Resource(scanSettingBindingGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ScanSettingBindings in namespace %q: %w", c.namespace, err)
	}

	result := make([]ScanSettingBinding, 0, len(list.Items))
	for _, item := range list.Items {
		ssb, parseErr := parseScanSettingBinding(item.Object)
		if parseErr != nil {
			// Skip malformed resources rather than aborting the whole list.
			continue
		}
		result = append(result, ssb)
	}
	return result, nil
}

// GetScanSetting fetches a named ScanSetting from the given namespace.
func (c *k8sClient) GetScanSetting(ctx context.Context, namespace, name string) (*ScanSetting, error) {
	obj, err := c.dynamic.Resource(scanSettingGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get ScanSetting %q in namespace %q: %w", name, namespace, err)
	}

	ss, err := parseScanSetting(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("parse ScanSetting %q: %w", name, err)
	}
	return ss, nil
}

// parseScanSettingBinding converts an unstructured map into a ScanSettingBinding.
func parseScanSettingBinding(obj map[string]interface{}) (ScanSettingBinding, error) {
	meta, _ := obj["metadata"].(map[string]interface{})
	name, _ := meta["name"].(string)
	namespace, _ := meta["namespace"].(string)

	spec, _ := obj["spec"].(map[string]interface{})

	// Parse profiles list into []NamedObjectReference.
	var profiles []NamedObjectReference
	if rawProfiles, ok := spec["profiles"].([]interface{}); ok {
		for _, rp := range rawProfiles {
			pm, ok := rp.(map[string]interface{})
			if !ok {
				continue
			}
			profiles = append(profiles, NamedObjectReference{
				Name:     stringField(pm, "name"),
				Kind:     stringField(pm, "kind"),
				APIGroup: stringField(pm, "apiGroup"),
			})
		}
	}

	// Parse settingsRef as a NamedObjectReference.
	var settingsRef *NamedObjectReference
	if sr, ok := spec["settingsRef"].(map[string]interface{}); ok {
		settingsRef = &NamedObjectReference{
			Name:     stringField(sr, "name"),
			Kind:     stringField(sr, "kind"),
			APIGroup: stringField(sr, "apiGroup"),
		}
	}

	if name == "" {
		return ScanSettingBinding{}, errors.New("ScanSettingBinding has no name")
	}

	// Populate ScanSettingName from settingsRef.Name for backward compatibility
	// with callers that read the flat field (e.g. mapping package).
	scanSettingName := ""
	if settingsRef != nil {
		scanSettingName = settingsRef.Name
	}

	return ScanSettingBinding{
		Namespace:       namespace,
		Name:            name,
		ScanSettingName: scanSettingName,
		SettingsRef:     settingsRef,
		Profiles:        profiles,
	}, nil
}

// parseScanSetting converts an unstructured map into a ScanSetting.
func parseScanSetting(obj map[string]interface{}) (*ScanSetting, error) {
	meta, _ := obj["metadata"].(map[string]interface{})
	name, _ := meta["name"].(string)
	namespace, _ := meta["namespace"].(string)

	// Schedule is nested under complianceSuiteSettings.schedule.
	schedule := ""
	if css, ok := obj["complianceSuiteSettings"].(map[string]interface{}); ok {
		schedule, _ = css["schedule"].(string)
	}

	if name == "" {
		return nil, errors.New("ScanSetting has no name")
	}

	return &ScanSetting{
		Namespace: namespace,
		Name:      name,
		Schedule:  schedule,
	}, nil
}

// stringField safely extracts a string value from an unstructured map.
func stringField(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

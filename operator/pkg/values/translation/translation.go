package translation

import (
	"fmt"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ResourcesKey is a key for most resources chart values.
	ResourcesKey = "resources"
	// TolerationsKey is the default tolerations key used in the charts.
	TolerationsKey = "tolerations"
)

// GetResources converts platform.Resources to chart values builder.
func GetResources(resources *corev1.ResourceRequirements) *ValuesBuilder {
	if resources == nil {
		return nil
	}
	res := NewValuesBuilder()

	if len(resources.Requests) > 0 {
		res.SetResourceList("requests", resources.Requests.DeepCopy())
	}
	if len(resources.Limits) > 0 {
		res.SetResourceList("limits", resources.Limits.DeepCopy())
	}
	return &res
}

// GetCustomize converts platform.CustomizeSpec to chart values builder.
func GetCustomize(customizeSpec *platform.CustomizeSpec) *ValuesBuilder {
	if customizeSpec == nil {
		return nil
	}

	res := NewValuesBuilder()
	res.SetStringMap("labels", customizeSpec.Labels)
	res.SetStringMap("annotations", customizeSpec.Annotations)
	envVarMap := make(map[string]interface{}, len(customizeSpec.EnvVars))
	for i := range customizeSpec.EnvVars {
		envVar := customizeSpec.EnvVars[i]
		if _, ok := envVarMap[envVar.Name]; ok {
			res.SetError(errors.Errorf("duplicate environment variable name %q", envVar.Name))
			return &res
		}

		// We need the content of the env var without the name for the Helm charts. We cannot set the name to "",
		// since it doesn't have an omitempty tag. We could create a `map[string]interface{}` with `Value` and
		// `ValueFrom` ported over, but that would break if Kubernetes ever adds to the corev1.EnvVar type.
		// Hence, rely on unstructured conversion.
		unstructuredEnvVar, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&envVar)
		if err != nil {
			res.SetError(errors.Wrapf(err, "failed parsing environment variable %q", envVar.Name))
			return &res
		}
		delete(unstructuredEnvVar, "name")
		envVarMap[envVar.Name] = unstructuredEnvVar
	}
	res.SetMap("envVars", envVarMap)
	return &res
}

// GetImagePullSecrets converts corev1.LocalObjectReference to a *ValuesBuilder with an "imagePullSecrets" field.
func GetImagePullSecrets(imagePullSecrets []platform.LocalSecretReference) *ValuesBuilder {
	res := NewValuesBuilder()
	if len(imagePullSecrets) > 0 {
		var ps []string
		for _, secret := range imagePullSecrets {
			ps = append(ps, secret.Name)
		}
		existing := NewValuesBuilder()
		existing.SetStringSlice("useExisting", ps)
		res.AddChild("imagePullSecrets", &existing)
	}
	return &res
}

// GetTLSConfigValues converts platform.TLSConfig to a *ValuesBuilder with an "additionalCAs" field.
func GetTLSConfigValues(tls *platform.TLSConfig) *ValuesBuilder {
	if tls == nil || len(tls.AdditionalCAs) == 0 {
		return nil
	}
	cas := NewValuesBuilder()
	for _, ca := range tls.AdditionalCAs {
		cas.SetStringValue(ca.Name, ca.Content)
	}
	res := NewValuesBuilder()
	res.AddChild("additionalCAs", &cas)
	return &res
}

// GetTolerations converts a slice of tolerations to a *ValuesBuilder object and sets the field name
// based on the key parameter.
func GetTolerations(key string, tolerations []*corev1.Toleration) *ValuesBuilder {
	v := NewValuesBuilder()

	var convertedList []interface{}
	for _, toleration := range tolerations {
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(toleration)
		if err != nil {
			v.SetError(errors.Wrapf(err, "failed converting %q to unstructured", key))
			break
		}
		convertedList = append(convertedList, m)
	}
	v.SetSlice(key, convertedList)

	return &v
}

// GetGlobalMonitoring converts *platform.GlobalMonitoring into *ValuesBuilder
func GetGlobalMonitoring(m *platform.GlobalMonitoring) *ValuesBuilder {
	openshiftMonitoring := NewValuesBuilder()
	// Default to true if undefined. Only set to false if explicitly disabled.
	openshiftMonitoring.SetBoolValue("enabled", !m.IsOpenShiftMonitoringDisabled())
	globalMonitoring := NewValuesBuilder()
	globalMonitoring.AddChild("openshift", &openshiftMonitoring)
	return &globalMonitoring
}

// SetScannerComponentDisabled sets the disabled values for scanner configurations
func SetScannerComponentDisabledValue(sv *ValuesBuilder, scannerComponent *platform.ScannerComponentPolicy) {
	if scannerComponent != nil {
		switch *scannerComponent {
		case platform.ScannerComponentDisabled:
			sv.SetBoolValue("disable", true)
		case platform.ScannerComponentEnabled:
			sv.SetBoolValue("disable", false)
		default:
			sv.SetError(fmt.Errorf("invalid spec.scanner.scannerComponent %q", *scannerComponent))
		}
	}
}

// SetScannerAnalyzerValues sets values in "sv" based on "analyzer".
func SetScannerAnalyzerValues(sv *ValuesBuilder, analyzer *platform.ScannerAnalyzerComponent) {
	if analyzer == nil {
		return
	}
	setScannerComponentScaling(sv, analyzer.GetScaling())
	sv.SetStringMap("nodeSelector", analyzer.NodeSelector)
	sv.AddChild(ResourcesKey, GetResources(analyzer.Resources))
	sv.AddAllFrom(GetTolerations(TolerationsKey, analyzer.DeploymentSpec.Tolerations))
}

// SetScannerDBValues sets values in "sv" based on "db".
func SetScannerDBValues(sv *ValuesBuilder, db *platform.DeploymentSpec) {
	if db != nil {
		sv.SetStringMap("dbNodeSelector", db.NodeSelector)
		sv.AddChild("dbResources", GetResources(db.Resources))
		sv.AddAllFrom(GetTolerations("dbTolerations", db.Tolerations))
	}
}

// SetScannerV4DBValues sets values in "sv" based on "db"
func SetScannerV4DBValues(sv *ValuesBuilder, db *platform.ScannerV4DB) {
	if db == nil {
		return
	}

	dbVB := NewValuesBuilder()
	dbVB.SetStringMap("nodeSelector", db.NodeSelector)
	dbVB.AddChild(ResourcesKey, GetResources(db.Resources))
	dbVB.AddAllFrom(GetTolerations(TolerationsKey, db.Tolerations))
	// TODO(ROX-19051): translate persistence values
	sv.AddChild("db", &dbVB)
	return
}

// SetScannerV4ComponentValues sets values in "sv" based on "component"
func SetScannerV4ComponentValues(sv *ValuesBuilder, componentKey string, component *platform.ScannerV4Component) {
	if component == nil {
		return
	}

	componentVB := NewValuesBuilder()
	setScannerComponentScaling(sv, component.Scaling)
	sv.SetStringMap("nodeSelector", component.NodeSelector)
	sv.AddChild(ResourcesKey, GetResources(component.Resources))
	sv.AddAllFrom(GetTolerations(TolerationsKey, component.Tolerations))
	sv.AddChild(componentKey, &componentVB)
}

func setScannerComponentScaling(sv *ValuesBuilder, scaling *platform.ScannerComponentScaling) {
	if scaling == nil {
		return
	}

	sv.SetInt32("replicas", scaling.Replicas)
	autoscalingVB := NewValuesBuilder()
	if scaling.AutoScaling != nil {
		switch *scaling.AutoScaling {
		case platform.ScannerAutoScalingDisabled:
			autoscalingVB.SetBoolValue("disable", true)
		case platform.ScannerAutoScalingEnabled:
			autoscalingVB.SetBoolValue("disable", false)
		default:
			autoscalingVB.SetError(fmt.Errorf("invalid scanner autoscaling %q", *scaling.AutoScaling))
		}
	}

	sv.SetInt32("maxReplicas", scaling.MaxReplicas)
	sv.SetInt32("minReplicas", scaling.MinReplicas)
	sv.AddChild("autoscaling", &autoscalingVB)
}

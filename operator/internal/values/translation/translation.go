package translation

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	utils "github.com/stackrox/rox/operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ResourcesKey is a key for most resources chart values.
	ResourcesKey = "resources"
	// TolerationsKey is the default tolerations key used in the charts.
	TolerationsKey = "tolerations"
	// HostAliasesKey is the default host aliases key used in the charts.
	HostAliasesKey = "hostAliases"

	defaultScannerV4PVCName = "scanner-v4-db"
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

// GetHostAliases converts a slice of host aliases to a *ValuesBuilder object and sets the field name
// based on the key parameter.
func GetHostAliases(key string, hostAliases []corev1.HostAlias) *ValuesBuilder {
	v := NewValuesBuilder()

	var convertedList []interface{}
	for i := range hostAliases {
		hostAlias := hostAliases[i]
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&hostAlias)
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

// SetScannerComponentDisableValue sets the value for the 'disable' key for scanner values
func SetScannerComponentDisableValue(sv *ValuesBuilder, scannerComponent *platform.ScannerComponentPolicy) {
	if scannerComponent == nil {
		return
	}

	switch *scannerComponent {
	case platform.ScannerComponentDisabled:
		sv.SetBoolValue("disable", true)
	case platform.ScannerComponentEnabled:
		sv.SetBoolValue("disable", false)
	default:
		sv.SetError(fmt.Errorf("invalid ScannerComponentPolicy %q", *scannerComponent))
	}
}

// SetScannerV4DisableValue sets the value for the 'disable' key for scanner values
func SetScannerV4DisableValue(sv *ValuesBuilder, scannerV4Component *platform.ScannerV4ComponentPolicy) {
	if scannerV4Component == nil {
		return
	}

	switch *scannerV4Component {
	case platform.ScannerV4ComponentDisabled:
		sv.SetBoolValue("disable", true)
	case platform.ScannerV4ComponentDefault:
		sv.SetBoolValue("disable", true)
	case platform.ScannerV4ComponentEnabled:
		sv.SetBoolValue("disable", false)
	default:
		sv.SetError(fmt.Errorf("invalid ScannerComponentPolicy %q", *scannerV4Component))
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
	if len(analyzer.HostAliases) > 0 {
		sv.AddAllFrom(GetHostAliases(HostAliasesKey, analyzer.HostAliases))
	}
}

// SetScannerDBValues sets values in "sv" based on "db".
func SetScannerDBValues(sv *ValuesBuilder, db *platform.DeploymentSpec) {
	if db != nil {
		sv.SetStringMap("dbNodeSelector", db.NodeSelector)
		sv.AddChild("dbResources", GetResources(db.Resources))
		sv.AddAllFrom(GetTolerations("dbTolerations", db.Tolerations))
		if len(db.HostAliases) > 0 {
			sv.AddAllFrom(GetHostAliases("dbHostAliases", db.HostAliases))
		}
	}
}

// SetScannerV4DBValues sets values in "sv" based on "db"
// In case of translating a secured cluster it checks for a default storage class
// if none is found it sets appropriate values to use an emptyDir.
// Unlike central-db's PVC we don't use the extension.ReconcilePVCExtension.
// The operator creates this PVC through the helm chart. This means it is managed
// by the default helm lifecycle, instead of the operator extension. The difference is
// that the extension prevents central DB's PVC deletion on deletion of the CR.
// Since Scanner V4's DB contains data which recovers by itself it is safe to remove the PVC
// through the helm uninstall if a CR is deleted.
func SetScannerV4DBValues(ctx context.Context, sv *ValuesBuilder, db *platform.ScannerV4DB, objKind string, namespace string, client ctrlClient.Reader) {
	dbVB := NewValuesBuilder()
	persistenceVB := NewValuesBuilder()

	useEmptyDir, err := shouldUseEmptyDir(ctx, db, objKind, namespace, client)
	if err != nil {
		sv.SetError(fmt.Errorf("error : %w", err))
		return
	}

	if useEmptyDir {
		persistenceVB.SetBoolValue("none", true)
		dbVB.AddChild("persistence", &persistenceVB)
		sv.AddChild("db", &dbVB)
		return
	}

	if db != nil {
		dbVB.SetStringMap("nodeSelector", db.NodeSelector)
		dbVB.AddChild(ResourcesKey, GetResources(db.Resources))
		dbVB.AddAllFrom(GetTolerations(TolerationsKey, db.Tolerations))
		if len(db.HostAliases) > 0 {
			dbVB.AddAllFrom(GetHostAliases(HostAliasesKey, db.HostAliases))
		}
	}

	setScannerV4DBPersistence(&dbVB, objKind, db.GetPersistence())
	sv.AddChild("db", &dbVB)
}

func setScannerV4DBPersistence(sv *ValuesBuilder, objKind string, persistence *platform.ScannerV4Persistence) {
	if persistence == nil {
		if objKind == platform.SecuredClusterGVK.Kind {
			// If no explicit config is set at this point set createClaim true
			// to explicitly signal to helm chart that we want a PVC
			// otherwise it will default to use emptyDir because it's
			// lookup for default StorageClass is turned off when using operator
			pvcBuilder := NewValuesBuilder()
			pvcBuilder.SetBool("createClaim", pointer.Bool(true))
			persistenceVB := NewValuesBuilder()
			persistenceVB.AddChild("persistentVolumeClaim", &pvcBuilder)
			sv.AddChild("persistence", &persistenceVB)
		}

		return
	}

	hostPath := persistence.GetHostPath()
	pvc := persistence.GetPersistentVolumeClaim()

	if hostPath != "" && pvc != nil {
		sv.SetError(errors.New("invalid persistence configuration, either hostPath or persistentVolumeClaim must be set, not both"))
		return
	}

	persistenceVB := NewValuesBuilder()
	if hostPath != "" {
		persistenceVB.SetStringValue("hostPath", hostPath)
	}

	if pvc != nil {
		pvcBuilder := NewValuesBuilder()
		pvcBuilder.SetString("claimName", pvc.ClaimName)
		pvcBuilder.SetBool("createClaim", pointer.Bool(true))
		pvcBuilder.SetString("storageClass", pvc.StorageClassName)
		pvcBuilder.SetString("size", pvc.Size)
		persistenceVB.AddChild("persistentVolumeClaim", &pvcBuilder)
	}

	sv.AddChild("persistence", &persistenceVB)
}

func shouldUseEmptyDir(ctx context.Context, db *platform.ScannerV4DB, objKind string, namespace string, client ctrlClient.Reader) (bool, error) {
	if objKind != v1alpha1.SecuredClusterGVK.Kind {
		return false, nil
	}

	pvcAlreadyExists, err := hasScannerV4DBPVC(ctx, client, db.GetPersistence().GetPersistentVolumeClaim().GetClaimName(), namespace)
	if err != nil {
		return false, err
	}

	if pvcAlreadyExists {
		return false, nil
	}

	storageClass := db.GetPersistence().GetPersistentVolumeClaim().GetStorageClassName()
	if storageClass != "" {
		return false, nil
	}

	hasSC, err := utils.HasDefaultStorageClass(ctx, client)
	if err != nil {
		return false, err
	}

	return !hasSC, nil
}

func hasScannerV4DBPVC(ctx context.Context, client ctrlClient.Reader, pvcName string, namespace string) (bool, error) {
	lookupPvc := pvcName
	if lookupPvc == "" {
		lookupPvc = defaultScannerV4PVCName
	}

	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      lookupPvc,
			Namespace: namespace,
		},
	}

	if err := client.Get(ctx, ctrlClient.ObjectKeyFromObject(&pvc), &pvc); err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("looking for existing scanner-v4-db pvc: %w", err)
	}

	return false, nil
}

// SetScannerV4ComponentValues sets values in "sv" based on "component"
func SetScannerV4ComponentValues(sv *ValuesBuilder, componentKey string, component *platform.ScannerV4Component) {
	if component == nil {
		return
	}

	componentVB := NewValuesBuilder()
	setScannerComponentScaling(&componentVB, component.Scaling)
	componentVB.SetStringMap("nodeSelector", component.NodeSelector)
	componentVB.AddChild(ResourcesKey, GetResources(component.Resources))
	componentVB.AddAllFrom(GetTolerations(TolerationsKey, component.Tolerations))
	if len(component.HostAliases) > 0 {
		componentVB.AddAllFrom(GetHostAliases(HostAliasesKey, component.HostAliases))
	}
	sv.AddChild(componentKey, &componentVB)
}

// DisableScannerV4Component produces Helm values that disable the provided Scanner V4 component
func DisableScannerV4Component(sv *ValuesBuilder, componentKey string) {
	componentVB := NewValuesBuilder()
	componentVB.SetBoolValue("disable", true)
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

	autoscalingVB.SetInt32("maxReplicas", scaling.MaxReplicas)
	autoscalingVB.SetInt32("minReplicas", scaling.MinReplicas)
	sv.AddChild("autoscaling", &autoscalingVB)
}

// GetGlobalNetwork converts *platform.GlobalNetworkSpec into *ValuesBuilder
func GetGlobalNetwork(s *platform.GlobalNetworkSpec) *ValuesBuilder {
	sv := NewValuesBuilder()
	if s.Policies != nil {
		sv.SetBoolValue("enableNetworkPolicies", s.IsNetworkPoliciesEnabled())
	}
	return &sv
}

// GetConfigAsCode converts a *platform.GetConfigAsCodeSpec into a *ValuesBuilder.
func GetConfigAsCode(c *platform.ConfigAsCodeSpec) *ValuesBuilder {
	sv := NewValuesBuilder()
	if c.ComponentPolicy != nil {
		sv.SetBoolValue("enabled", *c.ComponentPolicy == v1alpha1.ConfigAsCodeComponentEnabled)
	}
	return &sv
}

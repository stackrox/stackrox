package translation

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"

	// Required for the usage of go:embed below.
	_ "embed"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/securedcluster/scanner"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	sensorTLSSecretName           = "sensor-tls"
	admissionControlTLSSecretName = "admission-control-tls"
	collectorTLSSecretName        = "collector-tls"

	legacyCollectionKernelModule = "KernelModule"
)

var (
	//go:embed base-values.yaml
	baseValuesYAML []byte
)

// New creates a translator
func New(client ctrlClient.Client) Translator {
	return Translator{client: client}
}

// Translator translates and enriches helm values
type Translator struct {
	client ctrlClient.Client
}

// Translate translates and enriches helm values
func (t Translator) Translate(ctx context.Context, u *unstructured.Unstructured) (chartutil.Values, error) {
	baseValues, err := chartutil.ReadValues(baseValuesYAML)
	utils.CrashOnError(err) // ensured through unit test that this doesn't happen.

	sc := platform.SecuredCluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &sc)
	if err != nil {
		return nil, err
	}

	valsFromCR, err := t.translate(ctx, sc)
	if err != nil {
		return nil, err
	}

	imageOverrideVals, err := imageOverrides.ToValues()
	if err != nil {
		return nil, errors.Wrap(err, "computing image override values")
	}

	return helmUtil.CoalesceTables(baseValues, imageOverrideVals, valsFromCR), nil
}

// translate translates a SecuredCluster CR into helm values.
func (t Translator) translate(ctx context.Context, sc platform.SecuredCluster) (chartutil.Values, error) {
	t.setDefaults(&sc)

	v := translation.NewValuesBuilder()

	v.SetStringValue("clusterName", sc.Spec.ClusterName)
	v.SetStringMap("clusterLabels", sc.Spec.ClusterLabels)

	if sc.Spec.CentralEndpoint != "" {
		v.SetStringValue("centralEndpoint", sc.Spec.CentralEndpoint)
	}

	v.AddAllFrom(t.getTLSValues(ctx, sc))

	v.AddAllFrom(translation.GetImagePullSecrets(sc.Spec.ImagePullSecrets))

	customize := translation.NewValuesBuilder()

	scannerAutoSenseConfig, err := scanner.AutoSenseLocalScannerConfig(ctx, t.client, sc)
	if err != nil {
		return nil, err
	}

	v.AddChild("sensor", t.getSensorValues(sc.Spec.Sensor, scannerAutoSenseConfig))

	if sc.Spec.AdmissionControl != nil {
		v.AddChild("admissionControl", t.getAdmissionControlValues(sc.Spec.AdmissionControl))
	}

	if sc.Spec.AuditLogs != nil {
		v.AddChild("auditLogs", t.getAuditLogsValues(sc.Spec.AuditLogs))
	}

	if sc.Spec.PerNode != nil {
		v.AddChild("collector", t.getCollectorValues(sc.Spec.PerNode))
	}

	v.AddChild("scanner", t.getLocalScannerComponentValues(sc, scannerAutoSenseConfig))

	customize.AddAllFrom(translation.GetCustomize(sc.Spec.Customize))

	v.AddChild("customize", &customize)
	v.AddChild("meta", getMetaValues(sc))

	v.AddChild("monitoring", translation.GetGlobalMonitoring(sc.Spec.Monitoring))

	if sc.Spec.RegistryOverride != "" {
		v.SetStringValue("registryOverride", sc.Spec.RegistryOverride)
	}

	return v.Build()
}

// getTLSValues reads TLS configuration and looks up CA certificate from secrets.
func (t Translator) getTLSValues(ctx context.Context, sc platform.SecuredCluster) *translation.ValuesBuilder {
	v := translation.NewValuesBuilder()
	if err := t.checkRequiredTLSSecrets(ctx, sc); err != nil {
		return v.SetError(err)
	}

	v.SetBoolValue("createSecrets", false)
	sensorSecret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: sc.Namespace, Name: sensorTLSSecretName}
	if err := t.client.Get(ctx, key, sensorSecret); err != nil {
		return v.SetError(errors.Wrapf(err, "failed reading %q secret", sensorTLSSecretName))
	}

	centralCA, ok := sensorSecret.Data["ca.pem"]
	if !ok {
		return v.SetError(errors.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", sensorTLSSecretName))
	}
	v.SetStringMap("ca", map[string]string{"cert": string(centralCA)})

	v.AddAllFrom(translation.GetTLSConfigValues(sc.Spec.TLS))

	return &v
}

func (t Translator) checkRequiredTLSSecrets(ctx context.Context, sc platform.SecuredCluster) error {
	var finalErr error
	for _, name := range []string{sensorTLSSecretName, admissionControlTLSSecretName, collectorTLSSecretName} {
		if err := t.checkInitBundleSecret(ctx, sc, name); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}
	return finalErr
}

func (t Translator) checkInitBundleSecret(ctx context.Context, sc platform.SecuredCluster, secretName string) error {
	namespace := sc.Namespace
	secret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: namespace, Name: secretName}
	if err := t.client.Get(ctx, key, secret); err != nil {
		if k8sErrors.IsNotFound(err) {
			return errors.Wrapf(err, "init-bundle secret %q does not exist in namespace %q, please make sure you have downloaded init-bundle secrets (from UI or with roxctl) and created corresponding resources in the correct namespace", secretName, namespace)
		}
		return errors.Wrapf(err, "failed receiving secret %q", secretName)
	}
	return nil
}

func (t Translator) getSensorValues(sensor *platform.SensorComponentSpec, config scanner.AutoSenseResult) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()

	if sensor != nil {
		sv.AddChild(translation.ResourcesKey, translation.GetResources(sensor.Resources))
		sv.SetStringMap("nodeSelector", sensor.NodeSelector)
		sv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, sensor.Tolerations))
	}

	if config.EnableLocalImageScanning {
		sv.SetPathValue("localImageScanning.enabled", strconv.FormatBool(config.EnableLocalImageScanning))
	}

	return &sv
}

func (t Translator) getAdmissionControlValues(admissionControl *platform.AdmissionControlComponentSpec) *translation.ValuesBuilder {
	acv := translation.NewValuesBuilder()

	acv.AddChild(translation.ResourcesKey, translation.GetResources(admissionControl.Resources))
	acv.SetBool("listenOnCreates", admissionControl.ListenOnCreates)
	acv.SetBool("listenOnUpdates", admissionControl.ListenOnUpdates)
	acv.SetBool("listenOnEvents", admissionControl.ListenOnEvents)
	dynamic := translation.NewValuesBuilder()
	// Unlike in the UI, both static and dynamic parts of config are driven by
	// the single spec.admissionControl.listenOn* setting in CR. This is because
	// redeployment is natively part of the CR lifecycle when we have an operator, so
	// no need to distinguish between the static and dynamic part.
	dynamic.SetBool("enforceOnCreates", admissionControl.ListenOnCreates)
	dynamic.SetBool("enforceOnUpdates", admissionControl.ListenOnUpdates)
	if admissionControl.ContactImageScanners != nil {
		switch *admissionControl.ContactImageScanners {
		case platform.ScanIfMissing:
			dynamic.SetBoolValue("scanInline", true)
		case platform.DoNotScanInline:
			dynamic.SetBoolValue("scanInline", false)
		default:
			return dynamic.SetError(errors.Errorf("invalid spec.admissionControl.contactImageScanners setting %q", *admissionControl.ContactImageScanners))
		}
	}
	dynamic.SetInt32("timeout", admissionControl.TimeoutSeconds)
	if admissionControl.Bypass != nil {
		switch *admissionControl.Bypass {
		case platform.BypassBreakGlassAnnotation:
			dynamic.SetBoolValue("disableBypass", false)
		case platform.BypassDisabled:
			dynamic.SetBoolValue("disableBypass", true)
		default:
			return dynamic.SetError(errors.Errorf("invalid spec.admissionControl.bypass setting %q", *admissionControl.Bypass))
		}
	}
	acv.AddChild("dynamic", &dynamic)
	acv.SetStringMap("nodeSelector", admissionControl.NodeSelector)
	acv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, admissionControl.Tolerations))
	acv.SetInt32("replicas", admissionControl.Replicas)

	return &acv
}

func (t Translator) getAuditLogsValues(auditLogs *platform.AuditLogsSpec) *translation.ValuesBuilder {
	if auditLogs.Collection == nil || *auditLogs.Collection == platform.AuditLogsCollectionAuto {
		return nil
	}

	cv := translation.NewValuesBuilder()
	switch *auditLogs.Collection {
	case platform.AuditLogsCollectionEnabled:
		cv.SetBoolValue("disableCollection", false)
	case platform.AuditLogsCollectionDisabled:
		cv.SetBoolValue("disableCollection", true)
	default:
		return cv.SetError(errors.Errorf("invalid spec.auditLogs.collection setting %q", *auditLogs.Collection))
	}
	return &cv
}

func (t Translator) getCollectorValues(perNode *platform.PerNodeSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()

	if perNode.TaintToleration != nil {
		switch *perNode.TaintToleration {
		case platform.TaintTolerate:
			cv.SetBoolValue("disableTaintTolerations", false)
		case platform.TaintAvoid:
			cv.SetBoolValue("disableTaintTolerations", true)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.perNode.taintToleration %q", *perNode.TaintToleration))
		}
	}

	cv.AddAllFrom(t.getCollectorContainerValues(perNode.Collector))
	cv.AddAllFrom(t.getComplianceContainerValues(perNode.Compliance))
	cv.AddAllFrom(t.getNodeInventoryContainerValues(perNode.NodeInventory))

	return &cv
}

func (t Translator) getCollectorContainerValues(collectorContainerSpec *platform.CollectorContainerSpec) *translation.ValuesBuilder {
	if collectorContainerSpec == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()

	if c := collectorContainerSpec.Collection; c != nil {
		switch *c {
		case platform.CollectionEBPF:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_EBPF.String())
		case platform.CollectionNone:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_NO_COLLECTION.String())
		case platform.CollectionCOREBPF:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_CORE_BPF.String())
		case legacyCollectionKernelModule:
			// Kernel module collection has been removed, but for the
			// purposes of upgrades, we translate it to EBPF
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_EBPF.String())
		default:
			return cv.SetError(fmt.Errorf("invalid spec.perNode.collection %q", *c))
		}
	}

	if collectorContainerSpec.ImageFlavor != nil {
		switch *collectorContainerSpec.ImageFlavor {
		case platform.ImageFlavorSlim:
			cv.SetBoolValue("slimMode", true)
		case platform.ImageFlavorRegular:
			cv.SetBoolValue("slimMode", false)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.collector.collector.imageFlavor %q", *collectorContainerSpec.ImageFlavor))
		}
	}

	cv.AddChild(translation.ResourcesKey, translation.GetResources(collectorContainerSpec.Resources))

	return &cv
}

func (t Translator) getComplianceContainerValues(compliance *platform.ContainerSpec) *translation.ValuesBuilder {
	if compliance == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()
	cv.AddChild("complianceResources", translation.GetResources(compliance.Resources))

	return &cv
}

func (t Translator) getNodeInventoryContainerValues(nodeInventory *platform.ContainerSpec) *translation.ValuesBuilder {
	if nodeInventory == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()
	cv.AddChild("nodeScanningResources", translation.GetResources(nodeInventory.Resources))

	return &cv
}

func (t Translator) getLocalScannerComponentValues(securedCluster platform.SecuredCluster, config scanner.AutoSenseResult) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()
	s := securedCluster.Spec.Scanner

	sv.SetBoolValue("disable", !config.DeployScannerResources)

	translation.SetScannerAnalyzerValues(&sv, s.Analyzer)
	translation.SetScannerDBValues(&sv, s.DB)

	return &sv
}

// Sets defaults that might not be applied on the resource due to ROX-8046.
// Only defaults that result in behaviour different from the Helm chart defaults should be included here.
func (t Translator) setDefaults(sc *platform.SecuredCluster) {
	scanner.SetScannerDefaults(&sc.Spec)
	if sc.Spec.AdmissionControl == nil {
		sc.Spec.AdmissionControl = &platform.AdmissionControlComponentSpec{}
	}
	if sc.Spec.AdmissionControl.ListenOnCreates == nil {
		sc.Spec.AdmissionControl.ListenOnCreates = pointers.Bool(true)
	}
	if sc.Spec.AdmissionControl.ListenOnUpdates == nil {
		sc.Spec.AdmissionControl.ListenOnUpdates = pointers.Bool(true)
	}
}

func getMetaValues(sc platform.SecuredCluster) *translation.ValuesBuilder {
	meta := translation.NewValuesBuilder()
	fp, err := createConfigFingerprint(sc)
	if err != nil {
		return meta.SetError(err)
	}
	meta.SetStringValue("configFingerprintOverride", fp)
	return &meta
}

func createConfigFingerprint(sc platform.SecuredCluster) (string, error) {
	specAsYaml, err := yaml.Marshal(sc.Spec)
	if err != nil {
		return "", errors.Wrap(err, "marshaling SecuredCluster spec")
	}
	return fmt.Sprintf("%x", sha256.Sum256(specAsYaml)), nil
}

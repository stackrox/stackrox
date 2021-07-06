package translation

import (
	"context"
	"crypto/sha256"
	"fmt"

	// Required for the usage of go:embed below.
	_ "embed"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	securedcluster "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chartutil"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	supportedOperandNamespace     = "stackrox"
	supportedResourceName         = "stackrox-secured-cluster-services"
	sensorTLSSecretName           = "sensor-tls"
	admissionControlTLSSecretName = "admission-control-tls"
	collectorTLSSecretName        = "collector-tls"
)

var (
	//go:embed base-values.yaml
	baseValuesYAML []byte
)

// NewTranslator creates a translator
func NewTranslator(client kubernetes.Interface) Translator {
	return Translator{clientSet: client}
}

// Translator translates and enriches helm values
type Translator struct {
	clientSet kubernetes.Interface
}

// Translate translates and enriches helm values
func (t Translator) Translate(ctx context.Context, u *unstructured.Unstructured) (chartutil.Values, error) {
	baseValues, err := chartutil.ReadValues(baseValuesYAML)
	utils.CrashOnError(err) // ensured through unit test that this doesn't happen.

	sc := securedcluster.SecuredCluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &sc)
	if err != nil {
		return nil, err
	}

	if err := validate(sc); err != nil {
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

	return helmutil.CoalesceTables(baseValues, imageOverrideVals, valsFromCR), nil
}

// Translate translates a SecuredCluster CR into helm values.
func (t Translator) translate(ctx context.Context, sc securedcluster.SecuredCluster) (chartutil.Values, error) {
	v := translation.NewValuesBuilder()

	v.SetStringValue("clusterName", sc.Spec.ClusterName)

	v.SetStringValue("centralEndpoint", sc.Spec.CentralEndpoint)

	v.AddAllFrom(t.getTLSValues(ctx, sc))

	v.AddAllFrom(translation.GetImagePullSecrets(sc.Spec.ImagePullSecrets))

	// TODO(ROX-7178): support explicit env.openshift and env.istio setting

	customize := translation.NewValuesBuilder()

	if sc.Spec.Sensor != nil {
		v.AddChild("sensor", t.getSensorValues(sc.Spec.Sensor))
	}

	if sc.Spec.AdmissionControl != nil {
		v.AddChild("admissionControl", t.getAdmissionControlValues(sc.Spec.AdmissionControl))
	}

	if sc.Spec.PerNode != nil {
		v.AddChild("collector", t.getCollectorValues(sc.Spec.PerNode))
	}

	customize.AddAllFrom(translation.GetCustomize(sc.Spec.Customize))
	v.AddChild("customize", &customize)
	v.AddChild("meta", getMetaValues(sc))

	return v.Build()
}

func validate(sc securedcluster.SecuredCluster) error {
	if sc.Name != supportedResourceName {
		return errors.Errorf("invalid metadata.name: currently only %q is supported", supportedResourceName)
	}
	if sc.Namespace != supportedOperandNamespace {
		return errors.Errorf("invalid metadata.namespace: currently only %q is supported", supportedOperandNamespace)
	}
	return nil
}

// getTLSValues reads TLS configuration and looks up CA certificate from secrets.
func (t Translator) getTLSValues(ctx context.Context, sc securedcluster.SecuredCluster) *translation.ValuesBuilder {
	v := translation.NewValuesBuilder()
	if err := t.checkRequiredTLSSecrets(ctx, sc); err != nil {
		return v.SetError(err)
	}

	v.SetBoolValue("createSecrets", false)
	sensorSecret, err := t.clientSet.CoreV1().Secrets(sc.Namespace).Get(ctx, sensorTLSSecretName, metav1.GetOptions{})
	if err != nil {
		return v.SetError(errors.Wrapf(err, "failed reading %q secret", sensorTLSSecretName))
	}

	centralCA, ok := sensorSecret.Data["ca.pem"]
	if !ok {
		return v.SetError(errors.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", sensorTLSSecretName))
	}
	v.SetStringMap("ca", map[string]string{"cert": string(centralCA)})

	v.AddAllFrom(translation.GetTLSValues(sc.Spec.TLS))

	return &v
}

func (t Translator) checkRequiredTLSSecrets(ctx context.Context, sc securedcluster.SecuredCluster) error {
	var finalErr error
	for _, name := range []string{sensorTLSSecretName, admissionControlTLSSecretName, collectorTLSSecretName} {
		if err := t.checkInitBundleSecret(ctx, sc, name); err != nil {
			finalErr = multierror.Append(finalErr, err)
		}
	}
	return finalErr
}

func (t Translator) checkInitBundleSecret(ctx context.Context, sc securedcluster.SecuredCluster, secretName string) error {
	if _, err := t.clientSet.CoreV1().Secrets(sc.Namespace).Get(ctx, secretName, metav1.GetOptions{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			return errors.Wrapf(err, "init-bundle secret %q does not exist, please make sure you have downloaded init-bundle secrets (from UI or with roxctl) and created corresponding resources in the cluster", secretName)
		}
		return errors.Wrapf(err, "failed receiving secret %q", secretName)
	}
	return nil
}

func (t Translator) getSensorValues(sensor *securedcluster.SensorComponentSpec) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()

	sv.AddChild(translation.ResourcesKey, translation.GetResources(sensor.Resources))
	sv.SetStringMap("nodeSelector", sensor.NodeSelector)

	return &sv
}

func (t Translator) getAdmissionControlValues(admissionControl *securedcluster.AdmissionControlComponentSpec) *translation.ValuesBuilder {
	acv := translation.NewValuesBuilder()

	acv.AddChild(translation.ResourcesKey, translation.GetResources(admissionControl.Resources))
	acv.SetBool("listenOnCreates", admissionControl.ListenOnCreates)
	acv.SetBool("listenOnUpdates", admissionControl.ListenOnUpdates)
	acv.SetBool("listenOnEvents", admissionControl.ListenOnEvents)

	return &acv
}

func (t Translator) getCollectorValues(perNode *securedcluster.PerNodeSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()

	if perNode.TaintToleration != nil {
		switch *perNode.TaintToleration {
		case securedcluster.TaintTolerate:
			cv.SetBoolValue("disableTaintTolerations", false)
		case securedcluster.TaintAvoid:
			cv.SetBoolValue("disableTaintTolerations", true)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.perNode.taintToleration %q", *perNode.TaintToleration))
		}
	}

	cv.AddAllFrom(t.getCollectorContainerValues(perNode.Collector))
	cv.AddAllFrom(t.getComplianceContainerValues(perNode.Compliance))

	return &cv
}

func (t Translator) getCollectorContainerValues(collectorContainerSpec *securedcluster.CollectorContainerSpec) *translation.ValuesBuilder {
	if collectorContainerSpec == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()

	if c := collectorContainerSpec.Collection; c != nil {
		switch *c {
		case securedcluster.CollectionEBPF:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_EBPF.String())
		case securedcluster.CollectionKernelModule:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_KERNEL_MODULE.String())
		case securedcluster.CollectionNone:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_NO_COLLECTION.String())
		default:
			return cv.SetError(fmt.Errorf("invalid spec.perNode.collection %q", *c))
		}
	}

	if collectorContainerSpec.ImageFlavor != nil {
		switch *collectorContainerSpec.ImageFlavor {
		case securedcluster.ImageFlavorSlim:
			cv.SetBoolValue("slimMode", true)
		case securedcluster.ImageFlavorRegular:
			cv.SetBoolValue("slimMode", false)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.collector.collector.imageFlavor %q", *collectorContainerSpec.ImageFlavor))
		}
	}

	cv.AddChild(translation.ResourcesKey, translation.GetResources(collectorContainerSpec.Resources))

	return &cv
}

func (t Translator) getComplianceContainerValues(compliance *securedcluster.ContainerSpec) *translation.ValuesBuilder {
	if compliance == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()
	cv.AddChild("complianceResources", translation.GetResources(compliance.Resources))

	return &cv
}

func getMetaValues(sc securedcluster.SecuredCluster) *translation.ValuesBuilder {
	meta := translation.NewValuesBuilder()
	fp, err := createConfigFingerprint(sc)
	if err != nil {
		return meta.SetError(err)
	}
	meta.SetStringValue("configFingerprintOverride", fp)
	return &meta
}

func createConfigFingerprint(sc securedcluster.SecuredCluster) (string, error) {
	specAsYaml, err := yaml.Marshal(sc.Spec)
	if err != nil {
		return "", errors.Wrap(err, "marshaling SecuredCluster spec")
	}
	return fmt.Sprintf("%x", sha256.Sum256(specAsYaml)), nil
}

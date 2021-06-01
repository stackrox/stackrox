package translation

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	securedcluster "github.com/stackrox/rox/operator/api/securedcluster/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Translator translates and enriches helm values
type Translator struct {
	Config *rest.Config
}

// Translate translates and enriches helm values
func (t Translator) Translate(u *unstructured.Unstructured) (chartutil.Values, error) {
	sc := securedcluster.SecuredCluster{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &sc)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-7250): propagate context to Translators
	// TODO(ROX-7251): make sure that the client we create here is kosher
	return Translate(context.TODO(), kubernetes.NewForConfigOrDie(t.Config), sc)
}

// Translate translates a SecuredCluster CR into helm values.
func Translate(ctx context.Context, clientSet kubernetes.Interface, sc securedcluster.SecuredCluster) (chartutil.Values, error) {
	v := translation.NewValuesBuilder()

	// TODO(ROX-7125): prevent/allow cluster name change?
	v.SetStringValue("clusterName", sc.Spec.ClusterName)

	v.SetString("centralEndpoint", sc.Spec.CentralEndpoint)

	if sc.Spec.TLS != nil && len(sc.Spec.TLS.AdditionalCAs) > 0 {
		var cas []chartutil.Values
		for _, ca := range sc.Spec.TLS.AdditionalCAs {
			cas = append(cas, chartutil.Values{ca.Name: ca.Content})
		}
		v.SetChartutilValuesSlice("additionalCAs", cas)
	}

	// TODO(ROX-7179): support imagePullSecrets.allowNone and/or disabling fromDefaultServiceAccount?
	if len(sc.Spec.ImagePullSecrets) > 0 {
		var ps []string
		for _, secret := range sc.Spec.ImagePullSecrets {
			ps = append(ps, secret.Name)
		}
		v.SetChartutilValues("imagePullSecrets", chartutil.Values{"useExisting": ps})
	}

	// TODO(ROX-7178): support explicit env.openshift and env.istio setting
	// TODO(ROX-7148): support setting ca.cert
	// TODO(ROX-7150): support setting/overriding images

	customize := translation.NewValuesBuilder()

	if sc.Spec.Sensor != nil {
		v.AddChild("sensor", getSensorValues(ctx, clientSet, sc.Namespace, sc.Spec.Sensor))
		customize.AddChild("sensor", translation.GetCustomize(sc.Spec.Sensor.Customize))
	}

	if sc.Spec.AdmissionControl != nil {
		v.AddChild("admissionControl", getAdmissionControlValues(ctx, clientSet, sc.Namespace, sc.Spec.AdmissionControl))
		customize.AddChild("admission-control", translation.GetCustomize(sc.Spec.AdmissionControl.Customize))
	}

	if sc.Spec.Collector != nil {
		v.AddChild("collector", getCollectorValues(ctx, clientSet, sc.Namespace, sc.Spec.Collector))
		customize.AddChild("collector", translation.GetCustomize(sc.Spec.Collector.Customize))
	}

	customize.AddAllFrom(translation.GetCustomize(sc.Spec.Customize))
	v.AddChild("customize", &customize)

	return v.Build()
}

func getSensorValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, sensor *securedcluster.SensorComponentSpec) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()

	sv.SetPullPolicy("imagePullPolicy", sensor.ImagePullPolicy)
	sv.AddChild(translation.ResourcesKey, translation.GetResources(sensor.Resources))
	sv.AddAllFrom(translation.GetServiceTLS(ctx, clientSet, namespace, sensor.ServiceTLS))
	sv.SetStringMap("nodeSelector", sensor.NodeSelector)
	sv.SetString("endpoint", sensor.Endpoint)

	return &sv
}

func getAdmissionControlValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, admissionControl *securedcluster.AdmissionControlComponentSpec) *translation.ValuesBuilder {
	acv := translation.NewValuesBuilder()

	acv.SetPullPolicy("imagePullPolicy", admissionControl.ImagePullPolicy)
	acv.AddChild(translation.ResourcesKey, translation.GetResources(admissionControl.Resources))
	acv.AddAllFrom(translation.GetServiceTLS(ctx, clientSet, namespace, admissionControl.ServiceTLS))
	acv.SetBool("listenOnCreates", admissionControl.ListenOnCreates)
	acv.SetBool("listenOnUpdates", admissionControl.ListenOnUpdates)
	acv.SetBool("listenOnEvents", admissionControl.ListenOnEvents)

	return &acv
}

func getCollectorValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, collector *securedcluster.CollectorComponentSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()

	if collector.Collection != nil {
		switch *collector.Collection {
		case securedcluster.CollectionEBPF:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_EBPF.String())
		case securedcluster.CollectionKernelModule:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_KERNEL_MODULE.String())
		case securedcluster.CollectionNone:
			cv.SetStringValue("collectionMethod", storage.CollectionMethod_NO_COLLECTION.String())
		default:
			return cv.SetError(fmt.Errorf("invalid spec.collector.collection %q", *collector.Collection))
		}
	}

	if collector.TaintToleration != nil {
		switch *collector.TaintToleration {
		case securedcluster.TaintTolerate:
			cv.SetBoolValue("disableTaintTolerations", false)
		case securedcluster.TaintAvoid:
			cv.SetBoolValue("disableTaintTolerations", true)
		default:
			return cv.SetError(fmt.Errorf("invalid spec.collector.taintToleration %q", *collector.TaintToleration))
		}
	}

	cv.AddAllFrom(getCollectorContainerValues(collector.Collector))
	cv.AddAllFrom(getComplianceContainerValues(collector.Compliance))
	cv.AddAllFrom(translation.GetServiceTLS(ctx, clientSet, namespace, collector.ServiceTLS))

	return &cv
}

func getCollectorContainerValues(collectorContainerSpec *securedcluster.CollectorContainerSpec) *translation.ValuesBuilder {
	if collectorContainerSpec == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()

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

	cv.SetPullPolicy("imagePullPolicy", collectorContainerSpec.ImagePullPolicy)
	cv.AddChild(translation.ResourcesKey, translation.GetResources(collectorContainerSpec.Resources))

	// TODO(ROX-7176): make "customize" work for collector container
	return &cv
}

func getComplianceContainerValues(compliance *securedcluster.ContainerSpec) *translation.ValuesBuilder {
	if compliance == nil {
		return nil
	}

	cv := translation.NewValuesBuilder()
	cv.SetPullPolicy("complianceImagePullPolicy", compliance.ImagePullPolicy)
	cv.AddChild("complianceResources", translation.GetResources(compliance.Resources))

	// TODO(ROX-7176): make "customize" work for compliance container
	return &cv
}

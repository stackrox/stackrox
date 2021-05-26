package translation

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/operator/common/pkg/values/translation"
	securedcluster "github.com/stackrox/rox/operator/securedcluster/api/v1alpha1"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/client-go/kubernetes"
)

// Translate translates a SecuredCluster CR into helm values.
func Translate(ctx context.Context, clientSet kubernetes.Interface, sc securedcluster.SecuredCluster) (chartutil.Values, error) {
	v := chartutil.Values{}

	v["clusterName"] = sc.Spec.ClusterName
	// TODO(ROX-7125): prevent/allow cluster name change?

	if sc.Spec.CentralEndpoint != nil {
		v["centralEndpoint"] = *sc.Spec.CentralEndpoint
	}

	if sc.Spec.TLS != nil && len(sc.Spec.TLS.AdditionalCAs) > 0 {
		var cas []chartutil.Values
		for _, ca := range sc.Spec.TLS.AdditionalCAs {
			cas = append(cas, chartutil.Values{ca.Name: ca.Content})
		}
		v["additionalCAs"] = cas
	}

	// TODO(ROX-7179): support imagePullSecrets.allowNone and/or disabling fromDefaultServiceAccount?
	if len(sc.Spec.ImagePullSecrets) > 0 {
		var ps []string
		for _, secret := range sc.Spec.ImagePullSecrets {
			ps = append(ps, secret.Name)
		}
		v["imagePullSecrets"] = chartutil.Values{"useExisting": ps}
	}

	// TODO(ROX-7178): support explicit env.openshift and env.istio setting
	// TODO(ROX-7148): support setting ca.cert
	// TODO(ROX-7150): support setting/overriding images

	if err := setSensorValues(ctx, clientSet, sc.Namespace, v, sc.Spec.Sensor); err != nil {
		return nil, err
	}
	if err := setAdmissionControlValues(ctx, clientSet, sc.Namespace, v, sc.Spec.AdmissionControl); err != nil {
		return nil, err
	}
	if err := setCollectorValues(ctx, clientSet, sc.Namespace, v, sc.Spec.Collector); err != nil {
		return nil, err
	}

	translation.SetCustomize(sc.Spec.Customize, v, translation.CustomizeTopLevel)

	return v, nil
}

func setSensorValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, v chartutil.Values, sensor *securedcluster.SensorComponentSpec) error {
	if sensor == nil {
		return nil
	}
	sv := chartutil.Values{}
	if sensor.ImagePullPolicy != nil {
		sv["imagePullPolicy"] = *sensor.ImagePullPolicy
	}

	translation.SetResources(sensor.Resources, sv, translation.ResourcesLabel)

	err := translation.SetServiceTLS(ctx, clientSet, namespace, sensor.ServiceTLS, sv)
	if err != nil {
		return err
	}

	if sensor.NodeSelector != nil {
		sv["nodeSelector"] = sensor.NodeSelector
	}

	if sensor.Endpoint != nil {
		sv["endpoint"] = *sensor.Endpoint
	}

	translation.SetCustomize(sensor.Customize, v, translation.CustomizeSensor)

	if len(sv) > 0 {
		v["sensor"] = sv
	}
	return nil
}

func setAdmissionControlValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, v chartutil.Values, admissionControl *securedcluster.AdmissionControlComponentSpec) error {
	if admissionControl == nil {
		return nil
	}

	acv := chartutil.Values{}

	if admissionControl.ImagePullPolicy != nil {
		acv["imagePullPolicy"] = *admissionControl.ImagePullPolicy
	}

	translation.SetResources(admissionControl.Resources, acv, translation.ResourcesLabel)

	err := translation.SetServiceTLS(ctx, clientSet, namespace, admissionControl.ServiceTLS, acv)
	if err != nil {
		return err
	}

	translation.SetBool(admissionControl.ListenOnCreates, "listenOnCreates", acv)
	translation.SetBool(admissionControl.ListenOnUpdates, "listenOnUpdates", acv)
	translation.SetBool(admissionControl.ListenOnEvents, "listenOnEvents", acv)

	translation.SetCustomize(admissionControl.Customize, v, translation.CustomizeAdmissionControl)

	if len(acv) > 0 {
		v["admissionControl"] = acv
	}
	return nil
}

func setCollectorValues(ctx context.Context, clientSet kubernetes.Interface, namespace string, v chartutil.Values, collector *securedcluster.CollectorComponentSpec) error {
	if collector == nil {
		return nil
	}

	cv := chartutil.Values{}

	if collector.Collection != nil {
		switch *collector.Collection {
		case securedcluster.CollectionEBPF:
			cv["collectionMethod"] = storage.CollectionMethod_EBPF.String()
		case securedcluster.CollectionKernelModule:
			cv["collectionMethod"] = storage.CollectionMethod_KERNEL_MODULE.String()
		case securedcluster.CollectionNone:
			cv["collectionMethod"] = storage.CollectionMethod_NO_COLLECTION.String()
		default:
			return fmt.Errorf("invalid spec.collector.collection %q", *collector.Collection)
		}
	}

	if collector.TaintToleration != nil {
		switch *collector.TaintToleration {
		case securedcluster.TaintTolerate:
			cv["disableTaintTolerations"] = false
		case securedcluster.TaintAvoid:
			cv["disableTaintTolerations"] = true
		default:
			return fmt.Errorf("invalid spec.collector.taintToleration %q", *collector.TaintToleration)
		}
	}

	if err := setCollectorContainerValues(collector.Collector, cv); err != nil {
		return err
	}
	setComplianceContainerValues(collector.Compliance, cv)

	if err := translation.SetServiceTLS(ctx, clientSet, namespace, collector.ServiceTLS, cv); err != nil {
		return err
	}

	translation.SetCustomize(collector.Customize, v, translation.CustomizeCollector)

	if len(cv) > 0 {
		v["collector"] = cv
	}
	return nil
}

func setCollectorContainerValues(collectorContainerSpec *securedcluster.CollectorContainerSpec, cv chartutil.Values) error {
	if collectorContainerSpec == nil {
		return nil
	}
	if collectorContainerSpec.ImageFlavor != nil {
		switch *collectorContainerSpec.ImageFlavor {
		case securedcluster.ImageFlavorSlim:
			cv["slimMode"] = true
		case securedcluster.ImageFlavorRegular:
			cv["slimMode"] = false
		default:
			return fmt.Errorf("invalid spec.collector.collector.imageFlavor %q", *collectorContainerSpec.ImageFlavor)
		}
	}
	if collectorContainerSpec.ImagePullPolicy != nil {
		cv["imagePullPolicy"] = *collectorContainerSpec.ImagePullPolicy
	}
	translation.SetResources(collectorContainerSpec.Resources, cv, translation.ResourcesLabel)
	// TODO(ROX-7176): make "customize" work for collector container
	return nil
}

func setComplianceContainerValues(compliance *securedcluster.ContainerSpec, cv chartutil.Values) {
	if compliance == nil {
		return
	}
	if compliance.ImagePullPolicy != nil {
		cv["complianceImagePullPolicy"] = *compliance.ImagePullPolicy
	}
	translation.SetResources(compliance.Resources, cv, translation.ResourcesComplianceLabel)
	// TODO(ROX-7176): make "customize" work for compliance container
}

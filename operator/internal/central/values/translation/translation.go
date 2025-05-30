package translation

import (
	"context"
	"fmt"
	"strconv"

	// Required for the usage of go:embed below.
	_ "embed"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/common"
	"github.com/stackrox/rox/operator/internal/values/translation"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed base-values.yaml
	baseValuesYAML []byte
)

const (
	managedServicesAnnotation = "platform.stackrox.io/managed-services"
)

// New creates a new Translator
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

	c := platform.Central{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
	if err != nil {
		return nil, err
	}
	// For translation purposes, enrich Central with defaults, which are not implicitly marshalled/unmarshaled.
	if err := platform.AddUnstructuredDefaultsToCentral(&c, u); err != nil {
		return nil, err
	}

	// At this point we don't need the Defaults in the unstructured object anymore and simply get rid of it to prevent
	// Kube API warnings of the form:
	//
	//   KubeAPIWarningLogger    unknown field "defaults"
	delete(u.Object, "defaults")

	centralCopy := c.DeepCopy()
	valsFromCR, err := t.translate(ctx, *centralCopy)
	if err != nil {
		return nil, err
	}

	imageOverrideVals, err := imageOverrides.ToValues()
	if err != nil {
		return nil, errors.Wrap(err, "computing image override values")
	}

	return helmUtil.CoalesceTables(baseValues, imageOverrideVals, valsFromCR), nil
}

// translate translates a Central CR into helm values.
// This function potentially modifies the provided Central.
func (t Translator) translate(ctx context.Context, c platform.Central) (chartutil.Values, error) {
	if err := platform.MergeCentralDefaultsIntoSpec(&c); err != nil {
		return nil, err
	}

	v := translation.NewValuesBuilder()

	v.AddAllFrom(translation.GetImagePullSecrets(c.Spec.ImagePullSecrets))
	v.AddAllFrom(getEnv(c))
	v.AddAllFrom(translation.GetTLSConfigValues(c.Spec.TLS))

	customize := translation.NewValuesBuilder()
	customize.AddAllFrom(translation.GetCustomize(c.Spec.Customize))

	centralSpec := c.Spec.Central
	if centralSpec == nil {
		centralSpec = &platform.CentralComponentSpec{}
	}

	monitoring := c.Spec.Monitoring
	v.AddChild("monitoring", translation.GetGlobalMonitoring(monitoring))
	central, err := getCentralComponentValues(ctx, centralSpec, c.GetNamespace(), t.client)
	if err != nil {
		return nil, err
	}

	v.AddChild("central", central)

	if c.Spec.Scanner != nil {
		v.AddChild("scanner", getCentralScannerComponentValues(c.Spec.Scanner))
	}

	if c.Spec.ScannerV4 != nil {
		v.AddChild("scannerV4", getCentralScannerV4ComponentValues(ctx, c.Spec.ScannerV4, c.GetNamespace(), t.client))
	}

	v.AddChild("customize", &customize)

	if c.Spec.Network != nil {
		v.AddChild("network", translation.GetGlobalNetwork(c.Spec.Network))
	}

	if c.Spec.ConfigAsCode != nil {
		v.AddChild("configAsCode", translation.GetConfigAsCode(c.Spec.ConfigAsCode))
	}

	return v.Build()
}

func getEnv(c platform.Central) *translation.ValuesBuilder {
	env := translation.NewValuesBuilder()

	egress := c.Spec.Egress
	annotations := c.GetAnnotations()

	if egress != nil {
		if egress.ConnectivityPolicy != nil {
			switch *egress.ConnectivityPolicy {
			case platform.ConnectivityOnline:
				env.SetBoolValue("offlineMode", false)
			case platform.ConnectivityOffline:
				env.SetBoolValue("offlineMode", true)
			default:
				return env.SetError(fmt.Errorf("invalid spec.egress.connectivityPolicy %q", *egress.ConnectivityPolicy))
			}
		}
	}

	if annotations != nil {
		if annotationValue, ok := annotations[managedServicesAnnotation]; ok {
			managedServices, err := strconv.ParseBool(annotationValue)
			if err != nil {
				return env.SetError(fmt.Errorf("invalid annotation value %q for annotation %s",
					annotationValue, managedServicesAnnotation))
			}
			if managedServices {
				env.SetBoolValue("managedServices", true)
			}
		}
	}

	ret := translation.NewValuesBuilder()
	ret.AddChild("env", &env)
	return &ret
}

func getExposureRouteValues(r *platform.ExposureRoute) *translation.ValuesBuilder {
	if r == nil {
		return nil
	}
	route := translation.NewValuesBuilder()
	route.SetBool("enabled", r.Enabled)
	route.SetString("host", r.Host)
	if r.Reencrypt != nil {
		reencrypt := translation.NewValuesBuilder()
		reencrypt.SetBool("enabled", r.Reencrypt.Enabled)
		reencrypt.SetString("host", r.Reencrypt.Host)
		if r.Reencrypt.TLS != nil {
			tls := translation.NewValuesBuilder()
			tls.SetString("caCertificate", r.Reencrypt.TLS.CaCertificate)
			tls.SetString("certificate", r.Reencrypt.TLS.Certificate)
			tls.SetString("destinationCACertificate", r.Reencrypt.TLS.DestinationCACertificate)
			tls.SetString("key", r.Reencrypt.TLS.Key)
			reencrypt.AddChild("tls", &tls)
		}
		route.AddChild("reencrypt", &reencrypt)
	}
	return &route
}

func getCentralDBPersistenceValues(ctx context.Context, p *platform.DBPersistence, namespace string, client ctrlClient.Client) *translation.ValuesBuilder {
	persistence := translation.NewValuesBuilder()
	if hostPath := p.GetHostPath(); hostPath != "" {
		persistence.SetStringValue("hostPath", hostPath)
	} else {
		pvcBuilder := translation.NewValuesBuilder()
		pvcBuilder.SetBoolValue("createClaim", false)

		// Search for a backup PVC, if it exists, allow to mount it.
		backupClaimName := common.DefaultCentralDBBackupPVCName
		if pvc := p.GetPersistentVolumeClaim(); pvc != nil {
			if pvc.ClaimName != nil {
				backupClaimName = common.GetBackupClaimName(*pvc.ClaimName)
			}
			pvcBuilder.SetString("claimName", pvc.ClaimName)
		}

		key := ctrlClient.ObjectKey{
			Namespace: namespace,
			Name:      backupClaimName,
		}
		pvc := &corev1.PersistentVolumeClaim{}
		if err := client.Get(ctx, key, pvc); err == nil {
			persistence.SetBoolValue("_backup", true)
		} else {
			if !apiErrors.IsNotFound(err) {
				persistence.SetError(fmt.Errorf("could not find backup PVC, %w", err))
			}
		}

		persistence.AddChild("persistentVolumeClaim", &pvcBuilder)
	}
	return &persistence
}

func getCentralComponentValues(ctx context.Context, c *platform.CentralComponentSpec, namespace string, client ctrlClient.Client) (*translation.ValuesBuilder, error) {
	cv := translation.NewValuesBuilder()

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	if c.DefaultTLSSecret != nil {
		cv.SetMap("defaultTLS", map[string]interface{}{"reference": c.DefaultTLSSecret.Name})
	}

	cv.SetBoolValue("exposeMonitoring", c.Monitoring.IsEnabled())
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))

	if c.Exposure != nil {
		exposure := translation.NewValuesBuilder()
		if c.Exposure.LoadBalancer != nil {
			lb := translation.NewValuesBuilder()
			lb.SetBool("enabled", c.Exposure.LoadBalancer.Enabled)
			lb.SetInt32("port", c.Exposure.LoadBalancer.Port)
			lb.SetString("ip", c.Exposure.LoadBalancer.IP)
			exposure.AddChild("loadBalancer", &lb)
		}
		if c.Exposure.NodePort != nil {
			np := translation.NewValuesBuilder()
			np.SetBool("enabled", c.Exposure.NodePort.Enabled)
			np.SetInt32("port", c.Exposure.NodePort.Port)
			exposure.AddChild("nodePort", &np)
		}
		exposure.AddChild("route", getExposureRouteValues(c.Exposure.Route))
		cv.AddChild("exposure", &exposure)
	}

	if len(c.HostAliases) > 0 {
		cv.AddAllFrom(translation.GetHostAliases(translation.HostAliasesKey, c.HostAliases))
	}

	cv.AddChild("db", getCentralDBComponentValues(ctx, c.DB, namespace, client))
	cv.AddChild("telemetry", getTelemetryValues(c.Telemetry))

	cv.AddChild("declarativeConfiguration", getDeclarativeConfigurationValues(c.DeclarativeConfiguration))

	if c.GetNotifierSecretsEncryptionEnabled() {
		notifierSecretsEncryption := translation.NewValuesBuilder()
		notifierSecretsEncryption.SetBoolValue("enabled", true)
		cv.AddChild("notifierSecretsEncryption", &notifierSecretsEncryption)
	}

	return &cv, nil
}

func getCentralDBComponentValues(ctx context.Context, c *platform.CentralDBSpec, namespace string, client ctrlClient.Client) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()
	if c == nil {
		c = &platform.CentralDBSpec{}
	}

	if c.ConfigOverride.Name != "" {
		cv.SetStringValue("configOverride", c.ConfigOverride.Name)
	}

	source := translation.NewValuesBuilder()
	if c.ConnectionPoolSize != nil {
		source.SetInt32("minConns", c.ConnectionPoolSize.MinConnections)
		source.SetInt32("maxConns", c.ConnectionPoolSize.MaxConnections)
	}

	if c.ConnectionStringOverride != nil {
		if c.GetPersistence() != nil {
			cv.SetError(errors.New("if a connection string is provided, no persistence settings must be supplied"))
		}

		// TODO: there are other settings which are ignored in external mode - should we error if those are set, too?
		// Persistence seems fundamental, so it makes sense to error here, but a node selector can be regarded as more
		// accidental, that's why we tolerate it being specified. However, the reason we don't warn about it is mostly
		// that there is no good/easy way to warn.
		// Moreover, the behaviour of OpenShift console UI w.r.t. defaults is such that we cannot infer user intent
		// based merely on the (non-)nil-ness of a struct.
		// See https://github.com/stackrox/stackrox/pull/3322#discussion_r1005954280 for more details.

		cv.SetBoolValue("external", true)
		source.SetString("connectionString", c.ConnectionStringOverride)
		cv.AddChild("source", &source)
		return &cv
	}

	cv.AddChild("source", &source)
	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))
	cv.AddChild("persistence", getCentralDBPersistenceValues(ctx, c.GetPersistence(), namespace, client))
	if len(c.HostAliases) > 0 {
		cv.AddAllFrom(translation.GetHostAliases(translation.HostAliasesKey, c.HostAliases))
	}
	return &cv
}

func isTelemetryEnabled(t *platform.Telemetry) bool {
	if version.IsReleaseVersion() {
		// Enabled by default. Allow for empty key, as central may download it.
		return t == nil || t.Enabled == nil || *t.Enabled
	}
	// Disabled by default for development versions. But when enabled, allow
	// developers to configure telemetry for debugging purposes.
	// A key has to be provided though.
	return t != nil && t.Enabled != nil && *t.Enabled &&
		(t.Storage != nil && t.Storage.Key != nil && *t.Storage.Key != "")
}

func getTelemetryValues(t *platform.Telemetry) *translation.ValuesBuilder {
	if !isTelemetryEnabled(t) {
		tv := translation.NewValuesBuilder()
		tv.SetBoolValue("enabled", false)
		storage := translation.NewValuesBuilder()
		storage.SetString("key", pointer.String(phonehome.DisabledKey))
		tv.AddChild("storage", &storage)
		return &tv
	} else if t != nil && t.Storage != nil {
		tv := translation.NewValuesBuilder()
		tv.SetBoolValue("enabled", true)
		storage := translation.NewValuesBuilder()
		storage.SetString("key", t.Storage.Key)
		storage.SetString("endpoint", t.Storage.Endpoint)
		tv.AddChild("storage", &storage)
		return &tv
	}
	return nil
}

func getDeclarativeConfigurationValues(c *platform.DeclarativeConfiguration) *translation.ValuesBuilder {
	declarativeConfig := translation.NewValuesBuilder()
	if c == nil {
		return &declarativeConfig
	}

	mounts := translation.NewValuesBuilder()
	configMaps := make([]string, 0, len(c.ConfigMaps))
	secrets := make([]string, 0, len(c.Secrets))

	for _, cm := range c.ConfigMaps {
		configMaps = append(configMaps, cm.Name)
	}

	for _, secret := range c.Secrets {
		secrets = append(secrets, secret.Name)
	}

	mounts.SetStringSlice("configMaps", configMaps)
	mounts.SetStringSlice("secrets", secrets)
	declarativeConfig.AddChild("mounts", &mounts)
	return &declarativeConfig
}

func getCentralScannerComponentValues(s *platform.ScannerComponentSpec) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()

	translation.SetScannerComponentDisableValue(&sv, s.ScannerComponent)
	translation.SetScannerAnalyzerValues(&sv, s.GetAnalyzer())
	translation.SetScannerDBValues(&sv, s.DB)

	sv.SetBoolValue("exposeMonitoring", s.Monitoring.IsEnabled())

	return &sv
}

func getCentralScannerV4ComponentValues(ctx context.Context, s *platform.ScannerV4Spec, namespace string, client ctrlClient.Client) *translation.ValuesBuilder {
	sv := translation.NewValuesBuilder()
	translation.SetScannerV4DisableValue(&sv, s.ScannerComponent)
	translation.SetScannerV4ComponentValues(&sv, "indexer", s.Indexer)
	translation.SetScannerV4ComponentValues(&sv, "matcher", s.Matcher)
	translation.SetScannerV4DBValues(ctx, &sv, s.DB, platform.CentralGVK.Kind, namespace, client)

	if s.Monitoring != nil {
		sv.SetBoolValue("exposeMonitoring", s.Monitoring.IsEnabled())
	}

	return &sv
}

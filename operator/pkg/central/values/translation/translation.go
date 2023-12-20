package translation

import (
	"context"
	"fmt"
	"strconv"

	// Required for the usage of go:embed below.
	_ "embed"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/central/common"
	"github.com/stackrox/rox/operator/pkg/central/extensions"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"helm.sh/helm/v3/pkg/chartutil"
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

	valsFromCR, err := t.translate(ctx, c)
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
func (t Translator) translate(ctx context.Context, c platform.Central) (chartutil.Values, error) {
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

	obsoletePVC := common.ObsoletePVC(c.GetAnnotations())
	checker := &pvcStateChecker{
		ctx:       ctx,
		client:    t.client,
		namespace: c.GetNamespace(),
	}

	monitoring := c.Spec.Monitoring
	v.AddChild("monitoring", translation.GetGlobalMonitoring(monitoring))
	central, err := getCentralComponentValues(centralSpec, checker, obsoletePVC)
	if err != nil {
		return nil, err
	}

	v.AddChild("central", central)

	if c.Spec.Scanner != nil {
		v.AddChild("scanner", getCentralScannerComponentValues(c.Spec.Scanner))
	}

	v.AddChild("customize", &customize)

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

func getCentralDBPersistenceValues(p *platform.DBPersistence) *translation.ValuesBuilder {
	persistence := translation.NewValuesBuilder()
	if hostPath := p.GetHostPath(); hostPath != "" {
		persistence.SetStringValue("hostPath", hostPath)
	} else {
		pvcBuilder := translation.NewValuesBuilder()
		pvcBuilder.SetBoolValue("createClaim", false)
		if pvc := p.GetPersistentVolumeClaim(); pvc != nil {
			pvcBuilder.SetString("claimName", pvc.ClaimName)
		}

		persistence.AddChild("persistentVolumeClaim", &pvcBuilder)
	}
	return &persistence
}

func getCentralPersistenceValues(p *platform.Persistence, checker *pvcStateChecker, obsoletePVC bool) (*translation.ValuesBuilder, error) {
	persistence := translation.NewValuesBuilder()
	// Check pvcs which should exist in cluster.
	if obsoletePVC {
		persistence.SetBoolValue("none", true)
		return &persistence, nil
	}

	if hostPath := p.GetHostPath(); hostPath != "" {
		persistence.SetStringValue("hostPath", hostPath)
	} else {
		pvc := p.GetPersistentVolumeClaim()
		lookupName := extensions.DefaultCentralPVCName
		if pvc != nil {
			lookupName = pointer.StringDeref(pvc.ClaimName, extensions.DefaultCentralPVCName)
		}
		// Do not mount PVC if it does not exist.
		exists, err := checker.pvcExists(lookupName)
		if err != nil {
			return nil, err
		}
		if !exists {
			persistence.SetBoolValue("none", true)
			return &persistence, nil
		}

		pvcBuilder := translation.NewValuesBuilder()
		pvcBuilder.SetBoolValue("createClaim", false)
		if pvc != nil {
			pvcBuilder.SetString("claimName", pvc.ClaimName)
		}

		persistence.AddChild("persistentVolumeClaim", &pvcBuilder)
	}
	return &persistence, nil
}

func getCentralComponentValues(c *platform.CentralComponentSpec, checker *pvcStateChecker, obsoletePVC bool) (*translation.ValuesBuilder, error) {
	cv := translation.NewValuesBuilder()

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	if c.DefaultTLSSecret != nil {
		cv.SetMap("defaultTLS", map[string]interface{}{"reference": c.DefaultTLSSecret.Name})
	}

	cv.SetBoolValue("exposeMonitoring", c.Monitoring.IsEnabled())
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))

	// TODO(ROX-7147): design CentralEndpointSpec, see central_types.go

	persistence, err := getCentralPersistenceValues(c.GetPersistence(), checker, obsoletePVC)
	if err != nil {
		return nil, err
	}

	cv.AddChild("persistence", persistence)

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
		if c.Exposure.Route != nil {
			route := translation.NewValuesBuilder()
			route.SetBool("enabled", c.Exposure.Route.Enabled)
			route.SetString("host", c.Exposure.Route.Host)
			exposure.AddChild("route", &route)
		}
		cv.AddChild("exposure", &exposure)
	}

	cv.AddChild("db", getCentralDBComponentValues(c.DB))
	cv.AddChild("telemetry", getTelemetryValues(c.Telemetry))

	cv.AddChild("declarativeConfiguration", getDeclarativeConfigurationValues(c.DeclarativeConfiguration))

	if c.GetNotifierSecretsEncryptionEnabled() {
		notifierSecretsEncryption := translation.NewValuesBuilder()
		notifierSecretsEncryption.SetBoolValue("enabled", true)
		cv.AddChild("notifierSecretsEncryption", &notifierSecretsEncryption)
	}

	return &cv, nil
}

func getCentralDBComponentValues(c *platform.CentralDBSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()
	if c == nil {
		c = &platform.CentralDBSpec{}
	}

	if c.ConfigOverride.Name != "" {
		cv.SetStringValue("configOverride", c.ConfigOverride.Name)
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
		source := translation.NewValuesBuilder()
		source.SetString("connectionString", c.ConnectionStringOverride)
		cv.AddChild("source", &source)
		return &cv
	}

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))
	cv.AddChild("persistence", getCentralDBPersistenceValues(c.GetPersistence()))
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

	if s.ScannerComponent != nil {
		switch *s.ScannerComponent {
		case platform.ScannerComponentDisabled:
			sv.SetBoolValue("disable", true)
		case platform.ScannerComponentEnabled:
			sv.SetBoolValue("disable", false)
		default:
			return sv.SetError(fmt.Errorf("invalid spec.scanner.scannerComponent %q", *s.ScannerComponent))
		}
	}

	translation.SetScannerAnalyzerValues(&sv, s.GetAnalyzer())
	translation.SetScannerDBValues(&sv, s.DB)
	sv.SetBoolValue("exposeMonitoring", s.Monitoring.IsEnabled())

	return &sv
}

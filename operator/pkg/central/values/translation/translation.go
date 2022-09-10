package translation

import (
	"context"
	"fmt"
	"strconv"

	// Required for the usage of go:embed below.
	_ "embed"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stackrox/rox/pkg/features"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	//go:embed base-values.yaml
	baseValuesYAML []byte
)

const (
	managedServicesAnnotation = "platform.stackrox.io/managed-services"
)

// Translator translates and enriches helm values
type Translator struct {
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

	valsFromCR, err := translate(c)
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
func translate(c platform.Central) (chartutil.Values, error) {
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

	centralValues := getCentralComponentValues(centralSpec)
	if features.PostgresDatastore.Enabled() {
		if c.Spec.Central.DB == nil {
			return nil, errors.Errorf("%s is enabled, but no Central DB spec is specified", features.PostgresDatastore.EnvVar())
		}
		centralValues.AddChild("db", getCentralDBComponentValues(c.Spec.Central.DB))
	}
	v.AddChild("central", centralValues)

	if c.Spec.Scanner != nil {
		v.AddChild("scanner", getCentralScannerComponentValues(c.Spec.Scanner))
	}

	v.AddChild("customize", &customize)

	v.AddAllFrom(translation.GetMisc(c.Spec.Misc))

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

func getPersistenceValues(p *platform.Persistence) *translation.ValuesBuilder {
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

func getCentralComponentValues(c *platform.CentralComponentSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	if c.DefaultTLSSecret != nil {
		cv.SetMap("defaultTLS", map[string]interface{}{"reference": c.DefaultTLSSecret.Name})
	}

	cv.SetBoolValue("exposeMonitoring", c.Monitoring.IsEnabled())
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))

	// TODO(ROX-7147): design CentralEndpointSpec, see central_types.go

	cv.AddChild("persistence", getPersistenceValues(c.GetPersistence()))

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
	return &cv
}

func getCentralDBComponentValues(c *platform.CentralDBSpec) *translation.ValuesBuilder {
	cv := translation.NewValuesBuilder()
	cv.SetBoolValue("enabled", true)

	// Evaluate if a connection string is specified and if the operator should manage Central DB
	if c.ConnectionStringOverride != nil {
		cv.SetBoolValue("external", true)

		source := translation.NewValuesBuilder()
		source.SetString("connectionString", c.ConnectionStringOverride)
		cv.AddChild("source", &source)
		return &cv
	}

	cv.AddChild(translation.ResourcesKey, translation.GetResources(c.Resources))
	cv.SetStringMap("nodeSelector", c.NodeSelector)
	cv.AddAllFrom(translation.GetTolerations(translation.TolerationsKey, c.Tolerations))
	cv.AddChild("persistence", getPersistenceValues(c.GetPersistence()))
	return &cv
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

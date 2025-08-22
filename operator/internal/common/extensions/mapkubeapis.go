package extensions

import (
	"context"
	"fmt"
	"log"

	"github.com/go-logr/logr"
	mapkubeapisCommon "github.com/helm/helm-mapkubeapis/pkg/common"
	"github.com/helm/helm-mapkubeapis/pkg/mapping"
	mapkubeapisV3 "github.com/helm/helm-mapkubeapis/pkg/v3"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/internal/config/mapkubeapis"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddMapKubeAPIsExtensionIfMapFileExists conditionally adds the extension to opts if the map file exists
func AddMapKubeAPIsExtensionIfMapFileExists(opts []pkgReconciler.Option, mapper meta.RESTMapper) []pkgReconciler.Option {
	mapFile := mapkubeapis.GetMapFilePath()
	if mapFile == "" {
		ctrl.Log.Info("mapkubeapis map file does not exist, the extension will NOT be added")
		return opts
	}
	ctrl.Log.Info("mapkubeapis extension enabled", "mapFile", mapFile)

	config := MapKubeAPIsExtensionConfig{
		MapFile:    mapFile,
		RESTMapper: mapper,
		DubiousAPIs: []schema.GroupVersionKind{
			{
				Group:   "networking.istio.io",
				Version: "v1alpha3",
				Kind:    "DestinationRule",
			},
		},
	}
	extension := MapKubeAPIsExtension(config)
	return append(opts, pkgReconciler.WithPreExtension(extension))
}

// MapKubeAPIsExtensionConfig extension configuration
type MapKubeAPIsExtensionConfig struct {
	KubeConfig mapkubeapisCommon.KubeConfig
	MapFile    string
	// List of APIs which - if missing on the cluster - should be removed from release before processing,
	// to prevent helm upgrade errors such as:
	//   unable to build kubernetes objects from current release manifest: [resource mapping not found for name:
	//   "scanner-internal-no-istio -mtls" namespace: "stackrox" from "": no matches for kind "DestinationRule"
	//   in version "networking.istio.io/v1alpha3" ensure CRDs are installed first,
	DubiousAPIs []schema.GroupVersionKind
	RESTMapper  meta.RESTMapper
}

// MapKubeAPIsExtension checks the latest release version for any deprecated or removed APIs and performs
// the cleanup using helm mapkubeapis extension
func MapKubeAPIsExtension(config MapKubeAPIsExtensionConfig) extensions.ReconcileExtension {
	for _, api := range config.DubiousAPIs {
		if api.Group == "" || api.Version == "" || api.Kind == "" {
			log.Fatalf("MapKubeAPIsExtensionConfig dubious api %s group/version/kind is missing", api)
		}
	}
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), log logr.Logger) error {
		run := &mapKubeAPIsExtensionRun{
			ctx:           ctx,
			obj:           obj,
			statusUpdater: statusUpdater,
			log:           log,
			config:        config,
		}
		return run.Execute()
	}
}

type mapKubeAPIsExtensionRun struct {
	ctx           context.Context
	obj           *unstructured.Unstructured
	statusUpdater func(statusFunc extensions.UpdateStatusFunc)
	log           logr.Logger
	config        MapKubeAPIsExtensionConfig
}

func (r *mapKubeAPIsExtensionRun) Execute() error {
	const allK8sVersions = "v0.0.0"
	mapOptions := mapkubeapisCommon.MapOptions{
		ReleaseName:      r.obj.GetName(),
		ReleaseNamespace: r.obj.GetNamespace(),
		MapFile:          r.config.MapFile,
		KubeConfig:       r.config.KubeConfig,
	}

	var extra []*mapping.Mapping
	for _, dubiousGVK := range r.config.DubiousAPIs {
		if _, err := r.config.RESTMapper.RESTMapping(dubiousGVK.GroupKind(), dubiousGVK.Version); meta.IsNoMatchError(err) {
			r.log.Info("API not present, marking for removal from release.", "GVK", dubiousGVK, "error", err)
			extra = append(extra, &mapping.Mapping{
				DeprecatedAPI:    fmt.Sprintf("apiVersion: %s/%s\nkind: %s\n", dubiousGVK.Group, dubiousGVK.Version, dubiousGVK.Kind),
				RemovedInVersion: allK8sVersions,
			})
		}
	}

	if err := mapkubeapisV3.MapReleaseWithUnSupportedAPIs(mapOptions, extra...); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			r.log.V(1).Info("release not found, most likely it is not installed yet", "namespace", r.obj.GetNamespace(), "name", r.obj.GetName())
		} else {
			r.log.Error(err, "mapkubeapis run", "namespace", r.obj.GetNamespace(), "name", r.obj.GetName())
		}
	}

	return nil
}

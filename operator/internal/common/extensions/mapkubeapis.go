package extensions

import (
	"context"

	"github.com/go-logr/logr"
	mapkubeapisCommon "github.com/helm/helm-mapkubeapis/pkg/common"
	mapkubeapisV3 "github.com/helm/helm-mapkubeapis/pkg/v3"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	pkgReconciler "github.com/operator-framework/helm-operator-plugins/pkg/reconciler"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/config/mapkubeapis"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddMapKubeAPIsExtensionIfMapFileExists conditionally adds the extension to opts if the map file exists
func AddMapKubeAPIsExtensionIfMapFileExists(opts []pkgReconciler.Option) []pkgReconciler.Option {
	mapFile := mapkubeapis.GetMapFilePath()
	if mapFile == "" {
		ctrl.Log.Info("mapkubeapis map file does not exist, the extension will NOT be added")
		return opts
	}
	ctrl.Log.Info("mapkubeapis extension enabled", "mapFile", mapFile)

	config := MapKubeAPIsExtensionConfig{
		MapFile: mapFile,
	}
	extension := MapKubeAPIsExtension(config)
	return append(opts, pkgReconciler.WithPreExtension(extension))
}

// MapKubeAPIsExtensionConfig extension configuration
type MapKubeAPIsExtensionConfig struct {
	KubeConfig mapkubeapisCommon.KubeConfig
	MapFile    string
}

// MapKubeAPIsExtension checks the latest release version for any deprecated or removed APIs and performs
// the cleanup using helm mapkubeapis extension
func MapKubeAPIsExtension(config MapKubeAPIsExtensionConfig) extensions.ReconcileExtension {
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
	mapOptions := mapkubeapisCommon.MapOptions{
		ReleaseName:      r.obj.GetName(),
		ReleaseNamespace: r.obj.GetNamespace(),
		MapFile:          r.config.MapFile,
		KubeConfig:       r.config.KubeConfig,
	}

	if err := mapkubeapisV3.MapReleaseWithUnSupportedAPIs(mapOptions); err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			r.log.V(1).Info("release not found, most likely it is not installed yet", "namespace", r.obj.GetNamespace(), "name", r.obj.GetName())
		} else {
			r.log.Error(err, "mapkubeapis run", "namespace", r.obj.GetNamespace(), "name", r.obj.GetName())
		}
	}

	return nil
}

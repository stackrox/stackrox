package extensions

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	mapkubeapisCommon "github.com/helm/helm-mapkubeapis/pkg/common"
	mapkubeapisV3 "github.com/helm/helm-mapkubeapis/pkg/v3"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MapKubeAPIsExtensionConfig extension configuration
type MapKubeAPIsExtensionConfig struct {
	KubeConfig mapkubeapisCommon.KubeConfig
	MapFile    string
}

// MapKubeAPIsExtension checks the latest release version for any deprecated or removed APIs and performs
// // the cleanup using helm mapkubeapis extension
func MapKubeAPIsExtension() extensions.ReconcileExtension {
	configDir := os.Getenv("OPERATOR_CONFIG_DIR")
	if configDir == "" {
		configDir = "config"
	}
	config := MapKubeAPIsExtensionConfig{
		MapFile: filepath.Join(configDir, "mapkubeapis", "Map.yaml"),
	}
	return MapKubeAPIsExtensionWithConfig(config)
}

// MapKubeAPIsExtensionWithConfig checks the latest release version for any deprecated or removed APIs and performs
// the cleanup using helm mapkubeapis extension
func MapKubeAPIsExtensionWithConfig(config MapKubeAPIsExtensionConfig) extensions.ReconcileExtension {
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
	conditions    *[]platform.StackRoxCondition
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

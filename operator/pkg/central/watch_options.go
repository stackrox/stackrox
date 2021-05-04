/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package central

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/operator-sdk/helm/controller"
	"github.com/stackrox/rox/operator/pkg/operator-sdk/helm/release"
	"github.com/stackrox/rox/pkg/charts"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// CreateWatchOptions creates the watch options
func CreateWatchOptions(mgr manager.Manager) (controller.WatchOptions, error) {
	templateImage := image.GetDefaultImage()
	renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(image.CentralServicesChartPrefix, charts.RHACSMetaValues())
	if err != nil {
		return controller.WatchOptions{}, errors.Wrap(err, "loading and instantiating central services chart")
	}

	chart, err := loader.LoadFiles(renderedChartFiles)
	if err != nil {
		return controller.WatchOptions{}, errors.Wrap(err, "loading central services helm chart files")
	}
	return controller.WatchOptions{
		GVK:                     schema.GroupVersionKind{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Kind: "Central"}, //TODO: kind should be constant
		ManagerFactory:          release.NewManagerFactory(mgr, chart),
		WatchDependentResources: false, //TODO(do-not-merge): watching dependent resources?
		OverrideValues:          make(map[string]string),
	}, nil
}

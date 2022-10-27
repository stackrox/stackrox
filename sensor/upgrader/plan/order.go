package plan

import (
	"sort"

	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/common"
)

var (
	gvkPriorities = utils.InvertSlice(common.OrderedBundleResourceTypes)
)

func sortObjects[T k8sutil.Object](objects []T, reverse bool) {
	sort.Slice(objects, func(i, j int) bool {
		return reverse != (gvkPriorities[objects[i].GetObjectKind().GroupVersionKind()] < gvkPriorities[objects[j].GetObjectKind().GroupVersionKind()])
	})
}

func sortObjectRefs(objects []k8sobjects.ObjectRef, reverse bool) {
	sort.Slice(objects, func(i, j int) bool {
		return reverse != (gvkPriorities[objects[i].GVK] < gvkPriorities[objects[j].GVK])
	})
}

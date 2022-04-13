package plan

import (
	"sort"

	"github.com/stackrox/stackrox/pkg/k8sutil"
	"github.com/stackrox/stackrox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/sensor/upgrader/common"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	gvkPriorities = utils.Invert(common.OrderedBundleResourceTypes).(map[schema.GroupVersionKind]int)
)

func sortObjects(objects []k8sutil.Object, reverse bool) {
	sort.Slice(objects, func(i, j int) bool {
		return reverse != (gvkPriorities[objects[i].GetObjectKind().GroupVersionKind()] < gvkPriorities[objects[j].GetObjectKind().GroupVersionKind()])
	})
}

func sortObjectRefs(objects []k8sobjects.ObjectRef, reverse bool) {
	sort.Slice(objects, func(i, j int) bool {
		return reverse != (gvkPriorities[objects[i].GVK] < gvkPriorities[objects[j].GVK])
	})
}

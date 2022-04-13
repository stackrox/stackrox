package plan

import (
	"reflect"

	"github.com/stackrox/stackrox/pkg/k8sutil"
	"github.com/stackrox/stackrox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/stackrox/sensor/upgrader/common"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type planner struct {
	ctx      *upgradectx.UpgradeContext
	rollback bool
}

func (p *planner) objectsAreEqual(a, b k8sutil.Object) bool {
	var ua, ub unstructured.Unstructured
	if err := p.ctx.Scheme().Convert(a, &ua, nil); err != nil {
		return false
	}
	if err := p.ctx.Scheme().Convert(b, &ub, nil); err != nil {
		return false
	}

	normalizeObject(&ua)
	normalizeObject(&ub)

	return reflect.DeepEqual(ua.Object, ub.Object)
}

func (p *planner) GenerateExecutionPlan(desired []k8sutil.Object) (*ExecutionPlan, error) {
	currObjs, err := p.ctx.ListCurrentObjects()
	if err != nil {
		return nil, err
	}

	currObjMap := k8sobjects.BuildObjectMap(currObjs)

	var plan ExecutionPlan

	for _, desiredObj := range desired {
		ref := k8sobjects.RefOf(desiredObj)
		currObj := currObjMap[ref]

		log.Infof("Testing object %v", ref)
		if currObj == nil {
			plan.Creations = append(plan.Creations, desiredObj)
		} else {
			// We need to store this here because objectsAreEqual clobbers the resource version and annotations.
			currObjResourceVersion := currObj.GetResourceVersion()
			lastModifiedByThisProcessID := currObj.GetAnnotations()[common.LastUpgradeIDAnnotationKey] == p.ctx.ProcessID()

			if !p.rollback {
				newObj, err := applyPreservedProperties(p.ctx.Scheme(), desiredObj, currObj)
				if err != nil {
					log.Errorf("Failed to preserve properties for object %v: %v", ref, err)
				} else {
					desiredObj = newObj
				}
			}
			objectsAreEqual := p.objectsAreEqual(currObj, desiredObj)

			// We don't update if the objects are equal.
			// If the objects are not equal, we check if the object was already modified during this upgrade.
			// If it was, we skip the update.
			// If we're rolling back, though, we DO the update ONLY if it was modified during this upgrade.
			if !objectsAreEqual && lastModifiedByThisProcessID == p.rollback {
				desiredObj.SetResourceVersion(currObjResourceVersion)
				plan.Updates = append(plan.Updates, desiredObj)
			} else {
				log.Infof("Skipping update of object %v as it is unchanged or was already updated. Objects are equal: %v; last modified by this process ID: %v, rollback: %v",
					ref, objectsAreEqual, lastModifiedByThisProcessID, p.rollback)
			}
		}
		delete(currObjMap, ref)
	}

	for remainingObjRef := range currObjMap {
		plan.Deletions = append(plan.Deletions, remainingObjRef)
	}

	// sort objects such that dependency constraints are respected.
	sortObjects(plan.Creations, false)
	sortObjects(plan.Updates, false)
	sortObjectRefs(plan.Deletions, true)

	return &plan, nil
}

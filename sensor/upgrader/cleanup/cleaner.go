package cleanup

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/stackrox/pkg/kubernetes"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/sensor/upgrader/common"
	"github.com/stackrox/stackrox/sensor/upgrader/resources"
	"github.com/stackrox/stackrox/sensor/upgrader/upgradectx"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	log = logging.LoggerForModule()
)

type cleaner struct {
	ctx *upgradectx.UpgradeContext
}

func (c *cleaner) CleanupOwner() error {
	ownerRef := c.ctx.Owner()
	if ownerRef == nil {
		return errors.New("owner cleanup was requested, but no owner is known")
	}

	ownerResourceMD := c.ctx.GetResourceMetadata(ownerRef.GVK, 0)
	if ownerResourceMD == nil {
		return errors.Errorf("the cluster does not support the resource of the owning object %v", ownerRef)
	}

	client := c.ctx.DynamicClientForResource(ownerResourceMD, ownerRef.Namespace)
	return client.Delete(c.ctx.Context(), ownerRef.Name, kubernetes.DeleteBackgroundOption)
}

func (c *cleaner) CleanupState(own bool) error {
	var cmpOp string
	if own {
		cmpOp = "="
	} else {
		cmpOp = "!="
	}

	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s,%s!=,%s%s%s", common.UpgradeProcessIDLabelKey, common.UpgradeProcessIDLabelKey, common.UpgradeProcessIDLabelKey, cmpOp, c.ctx.ProcessID()),
	}

	stateObjs, err := c.ctx.List(resources.StateResource, &listOpts)
	if err != nil {
		return errors.Wrap(err, "listing upgrader state resources")
	}

	for _, obj := range stateObjs {
		log.Infof("Deleting leftover state object %v", k8sobjects.RefOf(obj))
		client, err := c.ctx.DynamicClientForGVK(obj.GetObjectKind().GroupVersionKind(), resources.StateResource, obj.GetNamespace())
		if err != nil {
			return err
		}
		if err := client.Delete(c.ctx.Context(), obj.GetName(), kubernetes.DeleteBackgroundOption); err != nil && !k8sErrors.IsNotFound(err) {
			return errors.Wrapf(err, "deleting %v", k8sobjects.RefOf(obj))
		}
	}
	return nil
}

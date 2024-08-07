package preflight

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/sensor/upgrader/plan"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"golang.org/x/exp/maps"
	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultServiceAccountName   = `sensor-upgrader`
	fallbackServiceAccountName  = `sensor`
	defaultClusterRoleBinding   = namespaces.StackRox + ":upgrade-sensors"
	upgraderTroubleshootingLink = "https://docs.openshift.com/acs/upgrading/upgrade-roxctl.html#upgrade-secured-clusters"
)

type accessCheck struct{}

func (accessCheck) Name() string {
	return "Kubernetes authorization"
}

func (accessCheck) getAllResourceAttribs(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan) ([]v1.ResourceAttributes, error) {
	resourceAttribsSet := make(map[v1.ResourceAttributes]struct{})

	for _, act := range execPlan.Actions() {
		var verb, rollbackVerb string
		switch act.ActionName {
		case plan.CreateAction:
			verb, rollbackVerb = "create", "delete"
		case plan.UpdateAction:
			verb = "update"
		case plan.DeleteAction:
			verb, rollbackVerb = "delete", "create"
		default:
			return nil, errors.Errorf("invalid action name %q for object %v", act.ActionName, act.ObjectRef)
		}

		resMD := ctx.GetResourceMetadata(act.ObjectRef.GVK, resources.BundleResource)
		if resMD == nil {
			return nil, errors.Errorf("no metadata available for resource %v", act.ObjectRef.GVK)
		}

		resourceAttribs := v1.ResourceAttributes{
			Namespace: act.ObjectRef.Namespace,
			Verb:      verb,
			Group:     resMD.Group,
			Version:   resMD.Version,
			Resource:  resMD.Name,
		}

		// Name only makes sense for update and delete
		if act.ActionName != plan.CreateAction {
			resourceAttribs.Name = act.ObjectRef.Name
		}

		resourceAttribsSet[resourceAttribs] = struct{}{}

		if rollbackVerb != "" {
			rollbackResourceAttribs := resourceAttribs
			// Name never makes sense for rollback as we'd be talking about objects that don't exist yet or will not
			// exist when we perform the rollback.
			rollbackResourceAttribs.Name = ""
			rollbackResourceAttribs.Verb = rollbackVerb
			resourceAttribsSet[rollbackResourceAttribs] = struct{}{}
		}
	}

	result := make([]v1.ResourceAttributes, 0, len(resourceAttribsSet))

	for resourceAttribs := range resourceAttribsSet {
		// TODO(mi): Sorting would be nice, but seems overkill
		result = append(result, resourceAttribs)
	}

	return result, nil
}

func (c accessCheck) Check(ctx *upgradectx.UpgradeContext, execPlan *plan.ExecutionPlan, reporter checkReporter) error {
	sarClient := ctx.ClientSet().AuthorizationV1().SelfSubjectAccessReviews()

	resourceAttribs, err := c.getAllResourceAttribs(ctx, execPlan)
	if err != nil {
		return err
	}

	actionResourceErr := make(map[string]struct{})
	for i := range resourceAttribs {
		ra := resourceAttribs[i]
		sar := &v1.SelfSubjectAccessReview{
			Spec: v1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &ra,
			},
		}
		sarResult, err := sarClient.Create(ctx.Context(), sar, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to perform SelfSubjectAccessReview check")
		}
		if sarResult.Status.EvaluationError != "" {
			reporter.Warnf("Evaluation error performing access review check for action %s on resource %s: %s", ra.Verb, ra.Resource, sarResult.Status.EvaluationError)
		}
		if !sarResult.Status.Allowed && !sarResult.Status.Denied {
			errEntry := fmt.Sprintf("%s:%s", ra.Verb, ra.Resource)
			actionResourceErr[errEntry] = struct{}{}
		} else if !sarResult.Status.Allowed {
			reporter.Errorf("Action %s on resource %s is not allowed", ra.Verb, ra.Resource)
		}
	}
	if len(actionResourceErr) > 0 {
		reporter.Errorf("This usually means that access is denied. "+
			"Have you configured this Secured Cluster for automatically receiving updates? "+
			"More info <a href=%q>More info on troubleshooting</a>.", upgraderTroubleshootingLink)

		affected := maps.Keys(actionResourceErr)
		slices.Sort(affected)
		log.Warnf("Kubernetes authorizer did not explicitly allow or deny "+
			"access to perform the following actions on the following resources: %s."+
			"This usually means that access is denied.", strings.Join(affected, ", "))
		issues := c.auxiliaryInfoOnPermissionDenied(ctx)
		for i, issue := range issues {
			log.Warnf("Discovered issue (%d of %d): %s", i+1, len(issues), issue)
		}
	}

	return nil
}

// auxiliaryInfoOnPermissionDenied returns string that should help understand why the upgrader does not have permission
// to run the upgrade. The string returned from this function will be displayed in the UI, so keep it brief.
func (c accessCheck) auxiliaryInfoOnPermissionDenied(ctx *upgradectx.UpgradeContext) []string {
	var msgs []string
	activeSA, err := c.getUpgraderSAName(ctx)
	if err != nil {
		log.Error(errors.Wrap(err, "failed to get upgrader SA name"))
		// unable to provide any additional info to the user if we don't know the SA
		return msgs
	}
	switch activeSA {
	case defaultServiceAccountName:
		if err := c.checkDefaultClusterRoleBinding(ctx, activeSA); err != nil {
			msgs = append(msgs, err.Error())
		}
	case "":
		msgs = append(msgs, fmt.Sprintf("Default ServiceAccount %q not found", defaultServiceAccountName))
		msgs = append(msgs, fmt.Sprintf("Fallback ServiceAccount %q not found", fallbackServiceAccountName))
	default:
		msgs = append(msgs, fmt.Sprintf("Upgrader is using %q ServiceAccount instead of the expected %q",
			activeSA, defaultServiceAccountName))
		if err := c.checkDefaultClusterRoleBinding(ctx, activeSA); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	return msgs
}

// getUpgraderSAName returns the name of the SA that is currently used by the upgrader
func (c accessCheck) getUpgraderSAName(ctx *upgradectx.UpgradeContext) (string, error) {
	deplClient := ctx.ClientSet().AppsV1().Deployments(namespaces.StackRox)
	depl, err := deplClient.Get(ctx.Context(), "sensor-upgrader", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return depl.Spec.Template.Spec.ServiceAccountName, nil
}

func (c accessCheck) checkDefaultClusterRoleBinding(ctx *upgradectx.UpgradeContext, saName string) error {
	crbClient := ctx.ClientSet().RbacV1().ClusterRoleBindings()
	crb, err := crbClient.Get(ctx.Context(), defaultClusterRoleBinding, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get default ClusterRoleBinding %q", defaultClusterRoleBinding)
	}
	if crb.RoleRef.Name != "cluster-admin" {
		return fmt.Errorf("ClusterRoleBinding %q is not bound to the cluster-admin ClusterRole", defaultClusterRoleBinding)
	}
	for _, subject := range crb.Subjects {
		if subject.Name == saName {
			return nil
		}
	}
	return fmt.Errorf("ClusterRoleBinding %q does not include subject %q ServiceAccount",
		defaultClusterRoleBinding, saName)
}

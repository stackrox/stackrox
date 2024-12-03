/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	configstackroxiov1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	"github.com/stackrox/rox/config-controller/pkg/client"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	policyFinalizer = "securitypolicies.config.stackrox.io/finalizer"
)

var (
	log = logging.LoggerForModule()
)

// SecurityPolicyReconciler reconciles a SecurityPolicy object
type SecurityPolicyReconciler struct {
	K8sClient     ctrlClient.Client
	Scheme        *runtime.Scheme
	CentralClient client.CachedCentralClient
}

//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/finalizers,verbs=update

func (r *SecurityPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Infof("Reconciling resource %q/%q", req.Namespace, req.Name)

	// Get the policy CR
	policyCR := &configstackroxiov1alpha1.SecurityPolicy{}
	if err := r.K8sClient.Get(ctx, req.NamespacedName, policyCR); err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("Failed to get policy: namespace=%s, name=%s", req.Namespace, req.Name)
	}

	if ok, err := policyCR.Spec.IsValid(); !ok {
		return ctrl.Result{}, errors.Wrapf(err, "Invalid policy resource: namespace=%s, name=%s", req.Namespace, req.Name)
	}

	if err := r.CentralClient.EnsureFresh(ctx); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to refresh")
	}

	desiredState := r.ToProtobuf(ctx, policyCR.Spec)

	existingPolicy, exists, err := r.CentralClient.GetPolicy(ctx, desiredState.GetName())
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to fetch policy")
	}

	// If the policy in CR is being renamed or does not exist on central, exists will be false, and we will update the policy ID
	// to the one in CR status. The policy ID in the CR status is expected to be blank if this is the first time policy is being reconciled.
	if exists {
		desiredState.Id = existingPolicy.GetId()
	} else {
		desiredState.Id = policyCR.Status.PolicyId
	}

	// Ensure a finalizer is added to each policy custom resource if it is not being deleted
	if policyCR.ObjectMeta.DeletionTimestamp.IsZero() {
		// The policy is not being deleted, so if it does not have our finalizer,
		// then lets update the policy to add the finalizer. This is equivalent
		// to registering our finalizer in preparation for a future delete.
		if !controllerutil.ContainsFinalizer(policyCR, policyFinalizer) {
			controllerutil.AddFinalizer(policyCR, policyFinalizer)
			if err := r.K8sClient.Update(ctx, policyCR); err != nil {
				return ctrl.Result{},
					errors.Wrapf(err, "failed to add finalizer to policy %q", policyCR.GetName())
			}
		}
	} else {
		// The policy is being deleted since k8s set the deletion timestamp
		if controllerutil.ContainsFinalizer(policyCR, policyFinalizer) {
			// finalizer is present, so lets handle the external dependency of deleting policy in central
			if policyCR.Status.Accepted {
				// Only try to delete a policy from Central if the CR has been marked as accepted
				if err := r.CentralClient.DeletePolicy(ctx, policyCR.Spec.PolicyName); err != nil {
					// if we failed to delete the policy in central, return with error
					// so that reconciliation can be retried.
					return ctrl.Result{}, errors.Wrapf(err, "failed to delete policy %q", policyCR.GetName())
				}
			}

			// delete on central was successful, so remove our finalizer from the list and update the resource.
			controllerutil.RemoveFinalizer(policyCR, policyFinalizer)
			if err := r.K8sClient.Update(ctx, policyCR); err != nil {
				return ctrl.Result{},
					errors.Wrapf(err, "failed to remove finalizer from policy %q", policyCR.GetName())
			}
		}
		// Stop reconciliation as the policy has been deleted
		return ctrl.Result{}, nil
	}

	if exists && existingPolicy.IsDefault {
		retErr := errors.New(fmt.Sprintf("Failed to reconcile: existing default policy with the same name '%s' exists", desiredState.GetName()))
		policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
			Accepted: false,
			Message:  retErr.Error(),
		}
		if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
			errMsg := fmt.Sprintf("error updating status for securitypolicy '%s'", policyCR.GetName())
			log.Debug(errMsg)
			return ctrl.Result{}, errors.Wrap(err, errMsg)
		}
		// We do not want this reconcile request to be requeued since it has a name collision
		// with an existing default policy hence return nil error.
		return ctrl.Result{}, nil
	}

	// policy create or update flow
	var retErr error
	if desiredState.GetId() != "" {
		log.Debugf("Updating policy %q (ID: %q)", desiredState.GetName(), desiredState.GetId())
		if err := r.CentralClient.UpdatePolicy(ctx, desiredState); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to update policy '%s'", desiredState.GetName()))
			policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
				Accepted: false,
				Message:  retErr.Error(),
			}
		} else {
			policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
				Accepted: true,
				Message:  "Successfully updated policy",
				PolicyId: desiredState.GetId(),
			}
		}
	} else {
		log.Debugf("Creating policy with name %q", desiredState.GetName())
		if createdPolicy, err := r.CentralClient.CreatePolicy(ctx, desiredState); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to create policy '%s'", desiredState.GetName()))
			policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
				Accepted: false,
				Message:  retErr.Error(),
			}
		} else {
			// Create was successful so persist the policy ID received from Central
			policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
				Accepted: true,
				Message:  "Successfully created policy",
				PolicyId: createdPolicy.GetId(),
			}
		}
	}

	if retErr != nil {
		// Perhaps the cache is stale, ignore errors since this is best effort
		_ = r.CentralClient.FlushCache(ctx)
	}

	if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
		errMsg := fmt.Sprintf("error updating status for securitypolicy %q", policyCR.GetName())
		log.Debug(errMsg)
		return ctrl.Result{}, errors.Wrap(err, errMsg)
	}

	return ctrl.Result{}, retErr
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&configstackroxiov1alpha1.SecurityPolicy{}).
		Complete(r)

	if err != nil {
		return errors.Wrap(err, "Failed to set up reconciler")
	}
	return nil
}

// ToProtobuf converts the SecurityPolicy spec into policy proto
func (r *SecurityPolicyReconciler) ToProtobuf(ctx context.Context, p configstackroxiov1alpha1.SecurityPolicySpec) *storage.Policy {
	proto := storage.Policy{
		Name:               p.PolicyName,
		Description:        p.Description,
		Rationale:          p.Rationale,
		Remediation:        p.Remediation,
		Disabled:           p.Disabled,
		Categories:         p.Categories,
		PolicyVersion:      policyversion.CurrentVersion().String(),
		CriteriaLocked:     p.CriteriaLocked,
		MitreVectorsLocked: p.MitreVectorsLocked,
		IsDefault:          p.IsDefault,
		Source:             storage.PolicySource_DECLARATIVE,
	}

	proto.Notifiers = make([]string, len(p.Notifiers))
	for _, notifier := range p.Notifiers {
		_, err := uuid.FromString(notifier)
		if err == nil {
			proto.Notifiers = append(proto.Notifiers, notifier)
			continue
		}
		// spec has notifier names specified
		id, exists := r.CentralClient.GetNotifierID(ctx, notifier)
		if exists {
			proto.Notifiers = append(proto.Notifiers, id)
			continue
		}
		log.Warnf("Notifier '%s' does not exist, skipping ..", notifier)
	}

	for _, ls := range p.LifecycleStages {
		val, found := storage.LifecycleStage_value[string(ls)]
		if found {
			proto.LifecycleStages = append(proto.LifecycleStages, storage.LifecycleStage(val))
		}
	}

	for _, exclusion := range p.Exclusions {
		protoExclusion := storage.Exclusion{
			Name: exclusion.Name,
		}

		if exclusion.Expiration != "" {
			protoTS, err := protocompat.ParseRFC3339NanoTimestamp(exclusion.Expiration)
			if err != nil {
				return nil
			}
			protoExclusion.Expiration = protoTS
		}

		if exclusion.Deployment != (configstackroxiov1alpha1.Deployment{}) {
			protoExclusion.Deployment = &storage.Exclusion_Deployment{
				Name: exclusion.Deployment.Name,
			}

			scope := exclusion.Deployment.Scope
			if scope != (configstackroxiov1alpha1.Scope{}) {
				protoExclusion.Deployment.Scope = &storage.Scope{
					Cluster:   scope.Cluster,
					Namespace: scope.Namespace,
				}
			}

			if scope.Label != (configstackroxiov1alpha1.Label{}) {
				protoExclusion.Deployment.Scope.Label = &storage.Scope_Label{
					Key:   scope.Label.Key,
					Value: scope.Label.Value,
				}
			}

		}

		proto.Exclusions = append(proto.Exclusions, &protoExclusion)
	}

	for _, scope := range p.Scope {
		protoScope := &storage.Scope{
			Cluster:   scope.Cluster,
			Namespace: scope.Namespace,
		}

		if scope.Label != (configstackroxiov1alpha1.Label{}) {
			protoScope.Label = &storage.Scope_Label{
				Key:   scope.Label.Key,
				Value: scope.Label.Value,
			}
		}

		proto.Scope = append(proto.Scope, protoScope)
	}

	val, found := storage.Severity_value[p.Severity]
	if found {
		proto.Severity = storage.Severity(val)
	}

	val, found = storage.EventSource_value[string(p.EventSource)]
	if found {
		proto.EventSource = storage.EventSource(val)
	}

	for _, ea := range p.EnforcementActions {
		val, found := storage.EnforcementAction_value[string(ea)]
		if found {
			proto.EnforcementActions = append(proto.EnforcementActions, storage.EnforcementAction(val))
		}
	}

	for _, section := range p.PolicySections {
		protoSection := &storage.PolicySection{
			SectionName: section.SectionName,
		}

		for _, group := range section.PolicyGroups {
			protoGroup := &storage.PolicyGroup{
				FieldName: group.FieldName,
				Negate:    group.Negate,
			}

			val, found = storage.BooleanOperator_value[group.BooleanOperator]
			if found {
				protoGroup.BooleanOperator = storage.BooleanOperator(val)
			}

			for _, value := range group.Values {
				protoValue := &storage.PolicyValue{
					Value: value.Value,
				}
				protoGroup.Values = append(protoGroup.Values, protoValue)
			}
			protoSection.PolicyGroups = append(protoSection.PolicyGroups, protoGroup)
		}
		proto.PolicySections = append(proto.PolicySections, protoSection)
	}

	for _, mitreAttackVectors := range p.MitreAttackVectors {
		protoMitreAttackVetor := &storage.Policy_MitreAttackVectors{
			Tactic:     mitreAttackVectors.Tactic,
			Techniques: mitreAttackVectors.Techniques,
		}

		proto.MitreAttackVectors = append(proto.MitreAttackVectors, protoMitreAttackVetor)
	}

	return &proto
}

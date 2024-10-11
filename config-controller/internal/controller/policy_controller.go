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
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	policyFinalizer = "securitypolicies.config.stackrox.io/finalizer"
)

// SecurityPolicyReconciler reconciles a SecurityPolicy object
type SecurityPolicyReconciler struct {
	K8sClient    ctrlClient.Client
	Scheme       *runtime.Scheme
	PolicyClient client.CachedPolicyClient
}

//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/finalizers,verbs=update

func (r *SecurityPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := log.FromContext(ctx)
	rlog.Info("Reconciling", "namespace", req.Namespace, "name", req.Name)

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

	if err := r.PolicyClient.EnsureFresh(ctx); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to refresh")
	}

	desiredState := policyCR.Spec.ToProtobuf()

	existingPolicy, exists, err := r.PolicyClient.GetPolicy(ctx, desiredState.GetName())
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to fetch policy")
	}

	if exists && existingPolicy.IsDefault {
		retErr := errors.New(fmt.Sprintf("Failed to reconcile: existing default policy with the same name '%s' exists", desiredState.GetName()))
		policyCR.Status = configstackroxiov1alpha1.SecurityPolicyStatus{
			Accepted: false,
			Message:  retErr.Error(),
		}
		if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
			rlog.Error(err, "error updating status for securitypolicy '%s'", policyCR.GetName())
			return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on security policy resource '%s'", policyCR.GetName()))
		}
		// We do not want this reconcile request to be requeued since it has a name collision
		// with an existing default policy hence return nil error.
		return ctrl.Result{}, nil
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
			if err := r.PolicyClient.DeletePolicy(ctx, policyCR.Spec.PolicyName); err != nil {
				// if we failed to delete the policy in central, return with error
				// so that reconciliation can be retried.
				return ctrl.Result{}, errors.Wrapf(err, "failed to delete policy %q", policyCR.GetName())
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

	// policy create or update flow
	var retErr error
	if desiredState.GetId() != "" {
		if err := r.PolicyClient.UpdatePolicy(ctx, desiredState); err != nil {
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
		if createdPolicy, err := r.PolicyClient.CreatePolicy(ctx, desiredState); err != nil {
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
		_ = r.PolicyClient.FlushCache(ctx)
	}

	if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
		rlog.Error(err, "error updating status for securitypolicy '%s'", policyCR.GetName())
		return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on security policy resource '%s'", policyCR.GetName()))
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

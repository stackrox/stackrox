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

	// Ensure a finalazer is added to each policy custom resource if it is not being deleted
	if policyCR.ObjectMeta.DeletionTimestamp.IsZero() {
		// The policy is not being deleted, so if it does not have our finalizer,
		// then lets update the policy to add the finalizer. This is equivalent
		// to registering our finalizer in preparation for a future delete.
		if !controllerutil.ContainsFinalizer(policyCR, policyFinalizer) {
			controllerutil.AddFinalizer(policyCR, policyFinalizer)
			if err := r.K8sClient.Update(ctx, policyCR); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "unable to add finalizer to policy %q", policyCR.GetName())
			}
		}
	} else {
		// The policy is being deleted since k8s set the deletion timestamp because a finalizer had already been added
		if controllerutil.ContainsFinalizer(policyCR, policyFinalizer) {
			// finalizer is present, so lets handle the external dependency of deleting policy in central
			if err := r.deletePolicyInCentral(ctx, policyCR); err != nil {
				// if we failed to delete the policy in central, return with error
				// so that it can be retried.
				return ctrl.Result{}, errors.Wrapf(err, "unable to delete policy %q", policyCR.GetName())

			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(policyCR, policyFinalizer)
			if err := r.K8sClient.Update(ctx, policyCR); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "unable to delete policy %q", policyCR.GetName())
			}
		} else {
			return ctrl.Result{}, errors.New(fmt.Sprintf("missing finalizer in policy %q", policyCR.GetName()))
		}

		// Stop reconciliation as the item has been deleted
		return ctrl.Result{}, nil
	}

	// Non deletion case, so either an update or create of a policy
	desiredState := policyCR.Spec.ToProtobuf()
	desiredState.Name = policyCR.GetName()
	desiredState.Source = storage.PolicySource_DECLARATIVE

	existingPolicy, exists, err := r.PolicyClient.GetPolicy(ctx, req.Name)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to fetch policy")
	}

	var retErr error
	if exists {
		desiredState.Id = existingPolicy.Id
		if err = r.PolicyClient.UpdatePolicy(ctx, desiredState); err != nil {
			desiredState.Id = existingPolicy.Id
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to update policy %s", req.Name))
			policyCR.Status.Accepted = false
			policyCR.Status.Message = retErr.Error()
		} else {
			policyCR.Status.Accepted = true
			policyCR.Status.Message = "Successfully updated policy"
		}
	} else {
		if _, err = r.PolicyClient.CreatePolicy(ctx, desiredState); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to create policy %s", req.Name))
			policyCR.Status.Accepted = false
			policyCR.Status.Message = retErr.Error()
		} else {
			policyCR.Status.Accepted = true
			policyCR.Status.Message = "Successfully created policy"
		}
	}

	if err = r.K8sClient.Status().Update(ctx, policyCR); err != nil {
		return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on policy %s", req.Name))
	}

	if retErr != nil {
		// Perhaps the cache is stale
		if err = r.PolicyClient.FlushCache(ctx); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Failed to flush cache")
		}
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

func (r *SecurityPolicyReconciler) deletePolicyInCentral(ctx context.Context, policyCR *configstackroxiov1alpha1.SecurityPolicy) error {
	rlog := log.FromContext(ctx)

	name := policyCR.GetName()
	cachedPolicy, exists, err := r.PolicyClient.GetPolicy(ctx, name)
	if !exists {
		rlog.Info("policy %q not found")
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "unable to find policy %q", name)
	}
	if cachedPolicy.GetSource() != storage.PolicySource_DECLARATIVE {
		return errors.New(fmt.Sprintf("policy %q is not externally managed and can be deleted only from central", name))
	}

	return errors.Wrap(r.PolicyClient.DeletePolicy(ctx, name),
		fmt.Sprintf("failed to delete policy %q in Central", name))
}

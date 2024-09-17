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
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	if err := r.PolicyClient.EnsureFresh(ctx); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to refresh")
	}

	// Get the policy CR
	policyCR := &configstackroxiov1alpha1.SecurityPolicy{}
	if err := r.K8sClient.Get(ctx, req.NamespacedName, policyCR); err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("Failed to get policy: namespace=%s, name=%s", req.Namespace, req.Name)
	}

	desiredState := policyCR.Spec.ToProtobuf()
	desiredState.Name = policyCR.GetName()

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
			retErr := errors.Wrap(err, fmt.Sprintf("Failed to create policy %s", req.Name))
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

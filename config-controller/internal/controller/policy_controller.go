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
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

	if len(policyCR.Status.Conditions) == 0 {
		policyCR.Status.Conditions = configstackroxiov1alpha1.SecurityPolicyConditions{
			configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:               configstackroxiov1alpha1.CentralDataFresh,
				Message:            "",
				Status:             "False",
				LastTransitionTime: metav1.Now(),
			},
			configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:               configstackroxiov1alpha1.PolicyValidated,
				Message:            "",
				Status:             "False",
				LastTransitionTime: metav1.Now(),
			},
			configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:               configstackroxiov1alpha1.AcceptedByCentral,
				Message:            "",
				Status:             "False",
				LastTransitionTime: metav1.Now(),
			},
		}
	}

	if result, refreshErr := r.UpdateCentralCaches(policyCR, ctx, r.CentralClient.EnsureFresh); refreshErr != nil {
		return result, refreshErr
	}

	desiredState, err := policyCR.Spec.ToProtobuf(map[configstackroxiov1alpha1.CacheType]map[string]string{
		configstackroxiov1alpha1.Notifier: r.CentralClient.GetNotifiers(),
		configstackroxiov1alpha1.Cluster:  r.CentralClient.GetClusters(),
	})
	if err != nil {
		retErr := errorhelpers.NewErrorList("")
		retErr.AddError(err)
		// This condition update will be persisted to the K8s API in the call to UpdateCentralCaches right below it
		policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
			Type:    configstackroxiov1alpha1.PolicyValidated,
			Status:  "False",
			Message: fmt.Sprintf("Unable to convert given spec to protobuf: %v", err),
		})
		// Attempt to refresh central caches and add error to error list if there was a problem updating the status of the CR
		if _, refreshErr := r.UpdateCentralCaches(policyCR, ctx, r.CentralClient.FlushCache); refreshErr != nil {
			retErr.AddError(err)
		}
		return ctrl.Result{}, errors.Wrap(retErr.ToError(), "Failed to convert policy to protobuf")
	}

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

		// finalizer is present, so lets handle the external dependency of deleting policy in central
		if controllerutil.ContainsFinalizer(policyCR, policyFinalizer) {
			// Only try to delete a policy from Central if the policy is active in Central
			if policyCR.Status.Conditions.IsAcceptedByCentral() && policyCR.Status.PolicyId != "" {
				if err := r.CentralClient.DeletePolicy(ctx, policyCR.Status.PolicyId); err != nil {
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
		policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
			Type:    configstackroxiov1alpha1.PolicyValidated,
			Status:  "False",
			Message: retErr.Error(),
		})
		if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
			errMsg := fmt.Sprintf("error updating status for securitypolicy '%s'", policyCR.GetName())
			log.Debug(errMsg)
			return ctrl.Result{}, errors.Wrap(err, errMsg)
		}
		// We do not want this reconcile request to be requeued since it has a name collision
		// with an existing default policy hence return nil error.
		return ctrl.Result{}, nil
	}

	policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
		Type:    configstackroxiov1alpha1.PolicyValidated,
		Status:  "True",
		Message: "Policy successfully validated.",
	})
	if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
		errMsg := fmt.Sprintf("error updating status for securitypolicy '%s'", policyCR.GetName())
		log.Debug(errMsg)
		return ctrl.Result{}, errors.Wrap(err, errMsg)
	}

	// policy create or update flow
	var retErr error
	if desiredState.GetId() != "" {
		log.Debugf("Updating policy %q (ID: %q)", desiredState.GetName(), desiredState.GetId())
		if err := r.CentralClient.UpdatePolicy(ctx, desiredState); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to update policy '%s'", desiredState.GetName()))
			policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:    configstackroxiov1alpha1.AcceptedByCentral,
				Status:  "False",
				Message: retErr.Error(),
			})
		} else {
			policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:    configstackroxiov1alpha1.AcceptedByCentral,
				Status:  "True",
				Message: "Policy was updated in Central.",
			})
			policyCR.Status.PolicyId = desiredState.GetId()
		}
	} else {
		log.Debugf("Creating policy with name %q", desiredState.GetName())
		if createdPolicy, err := r.CentralClient.CreatePolicy(ctx, desiredState); err != nil {
			retErr = errors.Wrap(err, fmt.Sprintf("Failed to create policy '%s'", desiredState.GetName()))
			policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:    configstackroxiov1alpha1.AcceptedByCentral,
				Status:  "False",
				Message: retErr.Error(),
			})
		} else {
			// Create was successful so persist the policy ID received from Central
			policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
				Type:    configstackroxiov1alpha1.AcceptedByCentral,
				Status:  "True",
				Message: "Policy was accepted by Central.",
			})
			policyCR.Status.PolicyId = createdPolicy.GetId()
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

func (r *SecurityPolicyReconciler) UpdateCentralCaches(policyCR *configstackroxiov1alpha1.SecurityPolicy, ctx context.Context, refreshFunc func(context.Context) error) (ctrl.Result, error) {
	if err := refreshFunc(ctx); err != nil {
		policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
			Type:    configstackroxiov1alpha1.CentralDataFresh,
			Status:  "False",
			Message: fmt.Sprintf("Unable to refresh Central caches: %v", err),
		})
	} else {
		policyCR.Status.Conditions.UpdateCondition(configstackroxiov1alpha1.SecurityPolicyCondition{
			Type:    configstackroxiov1alpha1.CentralDataFresh,
			Status:  "True",
			Message: "Central caches refreshed successfully.",
		})
	}
	if err := r.K8sClient.Status().Update(ctx, policyCR); err != nil {
		errMsg := fmt.Sprintf("Error updating status for SecurityPolicy '%s'", policyCR.GetName())
		log.Debug(errMsg)
		return ctrl.Result{}, errors.Wrap(err, errMsg)
	}
	return ctrl.Result{}, nil
}

func getEventFilter() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecurityPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&configstackroxiov1alpha1.SecurityPolicy{}).
		WithEventFilter(getEventFilter()).
		Complete(r)

	if err != nil {
		return errors.Wrap(err, "Failed to set up reconciler")
	}
	return nil
}

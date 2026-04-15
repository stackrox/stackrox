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
	"strconv"

	"github.com/pkg/errors"
	configstackroxiov1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	"github.com/stackrox/rox/config-controller/pkg/client"
	"github.com/stackrox/rox/generated/storage"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const riskPluginFinalizer = "riskscoringplugins.config.stackrox.io/finalizer"

// RiskScoringPluginReconciler reconciles a RiskScoringPlugin object
type RiskScoringPluginReconciler struct {
	K8sClient     ctrlClient.Client
	Scheme        *runtime.Scheme
	CentralClient client.CachedCentralClient
}

//+kubebuilder:rbac:groups=config.stackrox.io,resources=riskscoringplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.stackrox.io,resources=riskscoringplugins/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.stackrox.io,resources=riskscoringplugins/finalizers,verbs=update

func (r *RiskScoringPluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Infof("Reconciling RiskScoringPlugin %q/%q", req.Namespace, req.Name)

	// Get the CR
	pluginCR := &configstackroxiov1alpha1.RiskScoringPlugin{}
	if err := r.K8sClient.Get(ctx, req.NamespacedName, pluginCR); err != nil {
		if k8serr.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get RiskScoringPlugin %s/%s", req.Namespace, req.Name)
	}

	// Ensure token is fresh
	if err := r.CentralClient.EnsureFresh(ctx); err != nil {
		r.updateCondition(pluginCR, string(configstackroxiov1alpha1.RiskScoringPluginSynced),
			metav1.ConditionFalse, "TokenRefreshFailed", err.Error())
		if statusErr := r.K8sClient.Status().Update(ctx, pluginCR); statusErr != nil {
			log.Warnf("Failed to update status: %v", statusErr)
		}
		return ctrl.Result{}, errors.Wrap(err, "failed to refresh token")
	}

	// Handle deletion
	if !pluginCR.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(pluginCR, riskPluginFinalizer) {
			if pluginCR.Status.ConfigID != "" {
				if err := r.CentralClient.DeleteRiskScoringPluginConfig(ctx, pluginCR.Status.ConfigID); err != nil {
					return ctrl.Result{}, errors.Wrap(err, "failed to delete plugin config from Central")
				}
				log.Infof("Deleted plugin config %s from Central", pluginCR.Status.ConfigID)
			}
			controllerutil.RemoveFinalizer(pluginCR, riskPluginFinalizer)
			if err := r.K8sClient.Update(ctx, pluginCR); err != nil {
				return ctrl.Result{}, errors.Wrap(err, "failed to remove finalizer")
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(pluginCR, riskPluginFinalizer) {
		controllerutil.AddFinalizer(pluginCR, riskPluginFinalizer)
		if err := r.K8sClient.Update(ctx, pluginCR); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to add finalizer")
		}
	}

	// Convert CR to proto
	config, err := r.crToProto(pluginCR)
	if err != nil {
		r.updateCondition(pluginCR, string(configstackroxiov1alpha1.RiskScoringPluginSynced),
			metav1.ConditionFalse, "ValidationFailed", err.Error())
		if statusErr := r.K8sClient.Status().Update(ctx, pluginCR); statusErr != nil {
			log.Warnf("Failed to update status: %v", statusErr)
		}
		return ctrl.Result{}, errors.Wrap(err, "failed to convert CR to proto")
	}

	// Upsert to Central
	resp, err := r.CentralClient.UpsertRiskScoringPluginConfig(ctx, config)
	if err != nil {
		r.updateCondition(pluginCR, string(configstackroxiov1alpha1.RiskScoringPluginSynced),
			metav1.ConditionFalse, "SyncFailed", err.Error())
		if statusErr := r.K8sClient.Status().Update(ctx, pluginCR); statusErr != nil {
			log.Warnf("Failed to update status: %v", statusErr)
		}
		return ctrl.Result{}, errors.Wrap(err, "failed to upsert plugin config to Central")
	}

	// Update status
	pluginCR.Status.ConfigID = resp.GetId()
	r.updateCondition(pluginCR, string(configstackroxiov1alpha1.RiskScoringPluginSynced),
		metav1.ConditionTrue, "SyncSucceeded", "Plugin config synced to Central")

	if err := r.K8sClient.Status().Update(ctx, pluginCR); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to update status")
	}

	log.Infof("Successfully synced RiskScoringPlugin %s/%s (ConfigID: %s)", req.Namespace, req.Name, resp.GetId())
	return ctrl.Result{}, nil
}

func (r *RiskScoringPluginReconciler) crToProto(cr *configstackroxiov1alpha1.RiskScoringPlugin) (*storage.RiskScoringPluginConfig, error) {
	// Use namespace/name as ID for deterministic identification
	id := fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)

	// Parse weight string to float32
	weight := float32(1.0)
	if cr.Spec.Weight != "" {
		parsed, err := strconv.ParseFloat(cr.Spec.Weight, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid weight value %q", cr.Spec.Weight)
		}
		weight = float32(parsed)
	}

	// Validate builtin spec is present for builtin type
	if cr.Spec.Type == "builtin" && cr.Spec.Builtin == nil {
		return nil, errors.New("builtin spec is required when type is 'builtin'")
	}

	pluginName := ""
	var parameters map[string]string
	if cr.Spec.Builtin != nil {
		pluginName = cr.Spec.Builtin.Name
		parameters = cr.Spec.Builtin.Parameters
	}

	config := &storage.RiskScoringPluginConfig{
		Id:       id,
		Name:     pluginName,
		Type:     storage.PluginType_PLUGIN_TYPE_BUILTIN,
		Enabled:  cr.Spec.Enabled,
		Weight:   weight,
		Priority: cr.Spec.Priority,
		Builtin: &storage.BuiltinPluginConfig{
			PluginName: pluginName,
			Parameters: parameters,
		},
	}

	return config, nil
}

func (r *RiskScoringPluginReconciler) updateCondition(cr *configstackroxiov1alpha1.RiskScoringPlugin, condType string, status metav1.ConditionStatus, reason, message string) {
	cond := metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: cr.Generation,
	}
	meta.SetStatusCondition(&cr.Status.Conditions, cond)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RiskScoringPluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&configstackroxiov1alpha1.RiskScoringPlugin{}).
		WithEventFilter(getEventFilter()).
		Complete(r)

	if err != nil {
		return errors.Wrap(err, "failed to set up RiskScoringPlugin reconciler")
	}
	return nil
}

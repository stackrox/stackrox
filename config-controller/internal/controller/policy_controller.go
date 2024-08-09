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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/protoadapt"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	roxctlIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"

	configstackroxiov1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
)

// PolicyReconciler reconciles a Policy object
type PolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	conn   *grpc.ClientConn
}

//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.stackrox.io,resources=policies/finalizers,verbs=update

func (r *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rlog := log.FromContext(ctx)
	rlog.Info("Reconciling", "namespace", req.Namespace, "name", req.Name)

	// Get the policy CR
	policyCR := &configstackroxiov1alpha1.Policy{}
	if err := r.Client.Get(ctx, req.NamespacedName, policyCR); err != nil {
		if k8serr.IsNotFound(err) {
			// Must have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("Failed to get policy: namespace=%s, name=%s", req.Namespace, req.Name)
	}

	desiredState := &storage.Policy{}
	if err := protojson.Unmarshal([]byte(policyCR.Spec.Policy), protoadapt.MessageV2Of(desiredState)); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to parse policy JSON")
	}

	// GET policy from Central
	svc := v1.NewPolicyServiceClient(r.conn)
	allPolicies, err := svc.ListPolicies(ctx, &v1.RawQuery{})

	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to list all policies")
	}

	for _, policy := range allPolicies.Policies {
		if policy.Name == req.Name {
			desiredState.Id = policy.Id
			if _, err = svc.PutPolicy(ctx, desiredState); err != nil {
				return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to PUT policy %s", req.Name))
			}
			policyCR.Status.Accepted = true
			policyCR.Status.Message = "Successfully updated policy"
			if err = r.Client.Status().Update(ctx, policyCR); err != nil {
				return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on policy %s", req.Name))
			}
			return ctrl.Result{}, nil
		}
	}

	if _, err = svc.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: desiredState}); err != nil {
		wrappedErr := errors.Wrap(err, fmt.Sprintf("Failed to create policy %s", req.Name))
		policyCR.Status.Accepted = false
		policyCR.Status.Message = wrappedErr.Error()
		if err = r.Client.Status().Update(ctx, policyCR); err != nil {
			return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on policy %s", req.Name))
		}
		return ctrl.Result{}, wrappedErr
	} else {
		policyCR.Status.Accepted = true
		policyCR.Status.Message = "Successfully created policy"
		if err = r.Client.Status().Update(ctx, policyCR); err != nil {
			return ctrl.Result{}, errors.Wrap(err, fmt.Sprintf("Failed to set status on policy %s", req.Name))
		}
	}

	// Report status

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	conn, err := common.GetGRPCConnection(auth.TokenAuth(), logger.NewLogger(roxctlIO.DefaultIO(), printer.DefaultColorPrinter()))
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to Central")
	}
	r.conn = conn
	err = ctrl.NewControllerManagedBy(mgr).
		For(&configstackroxiov1alpha1.Policy{}).
		Complete(r)

	if err != nil {
		return errors.Wrap(err, "Failed to construct controller manager")
	}

	return nil
}

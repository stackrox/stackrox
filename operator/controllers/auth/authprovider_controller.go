/*
Copyright 2021 Red Hat.

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

package auth

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	authv1alpha1 "github.com/stackrox/rox/operator/apis/auth/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconciler reconciles a AuthProvider object
type Reconciler struct {
	ctrlClient.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=auth.stackrox.io,resources=authproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=auth.stackrox.io,resources=authproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=auth.stackrox.io,resources=authproviders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AuthProvider object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	authProvider := &authv1alpha1.AuthProvider{}
	if err := r.Client.Get(ctx, req.NamespacedName, authProvider); err != nil {
		if !apiErrors.IsNotFound(err) {
			return ctrl.Result{}, errors.Wrapf(err, "failed to get auth provider %s", authProvider.GetName())
		}
		return ctrl.Result{}, nil
	}

	// Check that the config map exists that represents the auth provider. If it doesn't exist, create a new one.
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, cm); err != nil && apiErrors.IsNotFound(err) {
		// Need to create the config map here.
		// This includes transforming it into a JSON representation from the CR.
		cm, err := r.configMapForAuthProvider(ctx, authProvider)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to create config map for auth provider")
		}

		if err := r.Create(ctx, cm); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to create config map")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to get configmap in namespace %s", req.NamespacedName)
	}

	// If the config map exists, ensure that the actual stored data is equivalent to the one we have received.
	authProviderMsgBytes, err := r.authProviderToProto(ctx, authProvider)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "creating auth provider JSON representation")
	}

	if !bytes.Equal(cm.BinaryData[authProvider.GetName()], authProviderMsgBytes) {
		cm.BinaryData[authProvider.GetName()] = authProviderMsgBytes
		if err := r.Update(ctx, cm); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to update configmap for auth provider %s", authProvider.GetName())
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) configMapForAuthProvider(ctx context.Context, authProvider *authv1alpha1.AuthProvider) (*corev1.ConfigMap, error) {
	// The transformation is the tricky part here.
	// CRs are identified by their name, whilst internally we use actual UUIDs to identify objects.
	// Somehow, we need to transform this here.
	// Whilst we could use the CR object UUID, this isn't really what you would expect from custom resources:
	// Re-creating the CR shouldn't lead to an entirely new object, but rather be consistent *iff* the same name is used.
	msgBytes, err := r.authProviderToProto(ctx, authProvider)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			// TODO(dhaus): Somehow need to make this one be dynamic and shard the auth providers across multiple config maps.
			// One could re-use this extraMounts helm setting to achieve this, but we have to check whether this is somethign we actually want to do or not.
			Name: "auth-providers",
			// TODO(dhaus): I suppose we can imagine that the CR will be in the same namespace as central, but for now hardcode it.
			Namespace: "stackrox",
		},
		BinaryData: map[string][]byte{
			authProvider.GetName(): msgBytes,
		},
	}
	if err := ctrl.SetControllerReference(authProvider, cm, r.Scheme); err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *Reconciler) authProviderToProto(ctx context.Context, authProvider *authv1alpha1.AuthProvider) ([]byte, error) {
	// Read from the secret reference.
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: authProvider.GetNamespace(),
		Name:      authProvider.Spec.ClientSecretReference.Name,
	}
	if err := r.Get(ctx, key, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to get secret %s for auth provider %s",
			authProvider.Spec.ClientSecretReference.Name, authProvider.GetName())
	}
	clientSecretB64 := secret.Data["client_secret"]
	var clientSecretRaw []byte
	n, err := base64.StdEncoding.Decode(clientSecretRaw, clientSecretB64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed decoding secret value from secret %s",
			secret.GetName())
	}
	clientSecret := string(clientSecretRaw[:n])

	provider := storage.AuthProvider{
		// For now, taking the short-cut of assigning ID == name. Can be done due to no explicit ID format + name uniqueness.
		// TODO(dhaus): For other types, this will be an issue and needs to be dealt with differently.
		Id:   authProvider.GetName(),
		Name: authProvider.GetName(),
		Type: string(*authProvider.Spec.Type),
		// TODO(dhaus): Not sure where this is getting set from.
		UiEndpoint: "",
		Enabled:    true,
		Config: map[string]string{
			"client_id":     *authProvider.Spec.ClientID,
			"client_secret": clientSecret,
			"issuer":        *authProvider.Spec.Issuer,
		},
		// For now, this is being set from the operator directly. Internally, this will only be set on creating the auth
		// provider within the service. Since we obviously cannot change this behavior, we have to set it for now within
		// here.
		LoginUrl:  "/sso/login/" + authProvider.GetName(),
		Validated: true,
		Active:    true,
		Traits: &storage.Traits{
			// TODO(dhaus): This should have the trait ALLOW_MUTATE_NEVER, meaning it won't be mutable even with providing force flag (i.e. never :)).
			MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
			Visibility:     storage.Traits_VISIBLE,
		},
	}

	return provider.Marshal()
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.AuthProvider{}).
		Complete(r)
}

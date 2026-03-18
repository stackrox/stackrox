package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type dockerConfigJSON struct {
	Auths map[string]dockerConfigEntry `json:"auths"`
}

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"` // notsecret
	Auth     string `json:"auth"`
}

func getDockerConfigSecret(namespace, id string, registryIdx int) *corev1.Secret {
	registry := fmt.Sprintf("registry-%04d.example.com", registryIdx)
	cfg := dockerConfigJSON{
		Auths: map[string]dockerConfigEntry{
			registry: {
				Username: "fake-user",
				Password: "fake-password", // notsecret
				Auth:     "ZmFrZS11c2VyOmZha2UtcGFzc3dvcmQ=",
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		log.Errorf("Failed to marshal docker config JSON: %v", err)
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pull-secret-%04d-%s", registryIdx, randStringWithLength(5)),
			Namespace: namespace,
			UID:       idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: data,
		},
	}
}

func getOpaqueSecret(namespace, id string, idx int) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("opaque-secret-%04d-%s", idx, randStringWithLength(5)),
			Namespace: namespace,
			UID:       idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte(randStringWithLength(64)),
		},
	}
}

func (w *WorkloadManager) getSecrets(workload SecretWorkload, ids []string) []runtime.Object {
	total := workload.NumDockerCfgSecrets + workload.NumOpaqueSecrets
	objects := make([]runtime.Object, 0, total)
	// Namespace assignment uses GetArbitraryElem which is hash-dependent, not
	// uniformly random. The goal is to trigger per-secret ResolveAllDeployments
	// amplification, not even distribution. If namespace imbalance matters for
	// a specific reproduction scenario, switch to round-robin.
	nsSet := make(map[string]struct{})
	idx := 0
	for i := 0; i < workload.NumDockerCfgSecrets; i++ {
		ns := namespacePool.mustGetRandomElem()
		nsSet[ns] = struct{}{}
		secret := getDockerConfigSecret(ns, getID(ids, idx), i)
		w.writeID(secretPrefix, secret.UID)
		objects = append(objects, secret)
		idx++
	}
	// Track namespaces that received docker config secrets so update waves
	// only scan relevant namespaces instead of the entire namespace pool.
	w.dockerSecretNamespaces = make([]string, 0, len(nsSet))
	for ns := range nsSet {
		w.dockerSecretNamespaces = append(w.dockerSecretNamespaces, ns)
	}
	for i := 0; i < workload.NumOpaqueSecrets; i++ {
		ns := namespacePool.mustGetRandomElem()
		secret := getOpaqueSecret(ns, getID(ids, idx), i)
		w.writeID(secretPrefix, secret.UID)
		objects = append(objects, secret)
		idx++
	}
	return objects
}

// manageSecrets periodically triggers waves of docker config secret updates.
// This simulates informer re-lists where all secrets are re-synced.
// Each wave updates ALL docker config secrets, and each update triggers
// ResolveAllDeployments() in the secret dispatcher — the amplification bomb.
// Early waves (before deployments sync) resolve to 0 deployments and are
// harmless; once deployments are in the store the amplification kicks in.
func (w *WorkloadManager) manageSecrets(ctx context.Context, workload SecretWorkload) {
	defer w.wg.Done()
	log.Infof("Secret workload: starting update waves every %s for %d docker config secrets",
		workload.UpdateInterval, workload.NumDockerCfgSecrets)

	ticker := time.NewTicker(workload.UpdateInterval)
	defer ticker.Stop()

	wave := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wave++
			w.updateAllDockerConfigSecrets(ctx, wave)
		}
	}
}

// updateAllDockerConfigSecrets touches every docker config secret in the fake
// client, triggering a SYNC-like wave through the secret informer. Each update
// causes the secret dispatcher to call ResolveAllDeployments().
func (w *WorkloadManager) updateAllDockerConfigSecrets(ctx context.Context, wave int) {
	updated := 0
	for _, ns := range w.dockerSecretNamespaces {
		secretClient := w.client.Kubernetes().CoreV1().Secrets(ns)
		list, err := secretClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		for i := range list.Items {
			secret := &list.Items[i]
			if secret.Type != corev1.SecretTypeDockerConfigJson && secret.Type != corev1.SecretTypeDockercfg {
				continue
			}
			if secret.Annotations == nil {
				secret.Annotations = make(map[string]string)
			}
			secret.Annotations["fake-workload/wave"] = fmt.Sprintf("%d", wave)
			if _, err := secretClient.Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				log.Errorf("error updating secret %s/%s: %v", secret.Namespace, secret.Name, err)
			}
			updated++
		}
	}
	log.Infof("Secret workload: wave %d updated %d docker config secrets", wave, updated)
}

package fake

import (
	"encoding/base64"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func getDockerConfigJSONSecret(id string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randStringWithLength(16),
			Namespace: namespacePool.mustGetRandomElem(),
			UID:       idOrNewUID(id),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{".dockerconfigjson": []byte(base64.StdEncoding.EncodeToString([]byte(randStringWithLength(16))))},
	}
}

func (w *WorkloadManager) getSecrets(workload SecretWorkload, dockerConfigJSONIDs []string) []runtime.Object {
	secrets := make([]runtime.Object, 0, workload.NumDockerConfigJSON)
	for i := 0; i < workload.NumDockerConfigJSON; i++ {
		log.Info("fake secret")
		secret := getDockerConfigJSONSecret(getID(dockerConfigJSONIDs, i))
		w.writeID(dockerConfigJSONSecretPrefix, secret.UID)
		secrets = append(secrets, secret)
	}
	return secrets
}

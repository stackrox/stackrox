package objects

import io.fabric8.kubernetes.api.model.ObjectReference

class K8sServiceAccount {
    String name
    String namespace
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    def automountToken
    List<ObjectReference> secrets = []
    String[] imagePullSecrets = []
}

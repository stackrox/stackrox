package objects

class K8sServiceAccount {
    String name
    String namespace
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    def automountToken
    List<io.fabric8.kubernetes.api.model.ObjectReference> secrets = []
    String[] imagePullSecrets = []
}

package objects

class K8sServiceAccount {
    def name
    def namespace
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    def automountToken
    def secrets = []
    def imagePullSecrets = []
}

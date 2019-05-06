package objects

class K8sRole {
    def name
    def namespace = ""
    def clusterRole = false
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    def rules = []
}

class K8sPolicyRule {
    def verbs
    def apiGroups
    def resources
    def nonResourceUrls
    def resourceNames
}

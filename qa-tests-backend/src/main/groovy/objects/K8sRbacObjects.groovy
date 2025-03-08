package objects

import groovy.transform.CompileStatic

@CompileStatic
class K8sRole {
    String name
    String namespace = ""
    def clusterRole = false
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    List<K8sPolicyRule> rules = []
    def uid
}

@CompileStatic
class K8sPolicyRule {
    List<String> verbs
    List<String> apiGroups
    List<String> resources
    List<String> nonResourceUrls
    List<String> resourceNames
}

@CompileStatic
class K8sRoleBinding  {
    String name
    String namespace
    Map<String, String> labels = [:]
    Map<String, String> annotations = [:]
    List<K8sSubject> subjects = []
    K8sRole roleRef

    K8sRoleBinding() {
    }

    K8sRoleBinding(K8sRole role, List<K8sSubject> subjects = []) {
        this.name = role.name
        this.namespace = role.namespace
        this.labels = role.labels
        this.annotations = role.annotations
        this.roleRef = role
        this.subjects = subjects
    }
}

@CompileStatic
class K8sSubject {
    def kind
    def name
    def namespace

    K8sSubject() {
    }

    K8sSubject(K8sServiceAccount serviceAccount) {
        this.kind = "ServiceAccount"
        this.name = serviceAccount.name
        this.namespace = serviceAccount.namespace
    }
}

package objects

class NetworkPolicy {
    String name
    String namespace
    Map<String, String> metadataPodSelector
    Map<String, String> ingressPodSelector
    Map<String, String> egressPodSelector
    Map<String, String> ingressNamespaceSelector
    Map<String, String> egressNamespaceSelector
    Set<NetworkPolicyTypes> types
    String uid

    NetworkPolicy(String name) {
        this.name = name
    }

    NetworkPolicy setNamespace(String namespace) {
        this.namespace = namespace
        return this
    }

    NetworkPolicy addPodSelector(Map<String, String> labels = [:]) {
        metadataPodSelector = metadataPodSelector ?: new HashMap<>()
        metadataPodSelector.putAll(labels)
        return this
    }

    NetworkPolicy addIngressPodSelector(Map<String, String> labels = [:]) {
        ingressPodSelector = ingressPodSelector ?: new HashMap<>()
        ingressPodSelector.putAll(labels)
        return this
    }

    NetworkPolicy addEgressPodSelector(Map<String, String> labels = [:]) {
        egressPodSelector = egressPodSelector ?: new HashMap<>()
        egressPodSelector.putAll(labels)
        return this
    }

    NetworkPolicy addIngressNamespaceSelector(Map<String, String> labels = [:]) {
        ingressNamespaceSelector = ingressNamespaceSelector ?: new HashMap<>()
        ingressNamespaceSelector.putAll(labels)
        return this
    }

    NetworkPolicy addEgressNamespaceSelector(Map<String, String> labels = [:]) {
        egressNamespaceSelector = egressNamespaceSelector ?: new HashMap<>()
        egressNamespaceSelector.putAll(labels)
        return this
    }

    NetworkPolicy addPolicyType(NetworkPolicyTypes type) {
        types = types ?: [] as Set
        types.add(type)
        return this
    }
}

enum NetworkPolicyTypes {
    INGRESS("Ingress"),
    EGRESS("Egress")

    private final String value

    NetworkPolicyTypes(String value) {
        this.value = value
    }

    @Override
    String toString() {
        return this.value
    }
}

package objects

import common.Constants

class Service {
    String name
    String namespace
    Map<String, String> labels = [:]
    Map<Integer, String> ports = [:]
    Type type = Type.CLUSTERIP
    Integer targetport
    String loadBalancerIP = null

    Service(String name, String namespace = Constants.ORCHESTRATOR_NAMESPACE) {
        this.name = name
        this.namespace = namespace
    }

    Service(Deployment deployment) {
        this.name = deployment.serviceName ?: deployment.name
        this.namespace = deployment.namespace
        this.labels = deployment.labels
        this.ports = deployment.ports
        this.type = deployment.createLoadBalancer ? Type.LOADBALANCER : Type.CLUSTERIP
        this.targetport = deployment.targetport
    }

    def addPort(int port, String type = "TCP") {
        ports.put(port, type)
        return this
    }

    def addLabel(String label, String value) {
        labels.put(label, value)
        return this
    }

    def setType(Type type) {
        this.type = type
        return this
    }

    def setTargetPort(Integer targetPort) {
        this.targetport = targetPort
        return this
    }

    enum Type {
        CLUSTERIP("ClusterIP"),
        LOADBALANCER("LoadBalancer"),
        NODEPORT("NodePort")

        private final String value

        Type(String value) {
            this.value = value
        }

        String toString() {
            return value
        }
    }
}

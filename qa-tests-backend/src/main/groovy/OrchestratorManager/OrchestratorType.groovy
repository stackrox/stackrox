package OrchestratorManager

/**
 * Created by parulshukla on 5/22/18.
 */
class OrchestratorType {
    static OrchestratorMain orchestrator

    static OrchestratorMain create(OrchestratorTypes type, String namespace = null) {
        switch (type) {
            case OrchestratorTypes.K8S:
                orchestrator = new Kubernetes(namespace)
                return orchestrator
                break

            case OrchestratorTypes.DDC2:
                orchestrator = new DockerEE()
                return orchestrator
                break

            case OrchestratorTypes.OPENSHIFT:
                orchestrator = new OpenShift()
                return orchestrator
                break

        }
    }
}


enum OrchestratorTypes {
    K8S,
    DDC2,
    OPENSHIFT,
    SWARM
}


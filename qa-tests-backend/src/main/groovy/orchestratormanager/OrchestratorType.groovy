package orchestratormanager

import groovy.transform.CompileStatic

/**
 * Created by parulshukla on 5/22/18.
 */
@CompileStatic
class OrchestratorType {
    static Kubernetes orchestrator

    static Kubernetes create(OrchestratorTypes type, String namespace = null) {
        switch (type) {
            case OrchestratorTypes.K8S:
                orchestrator = new Kubernetes(namespace)
                return orchestrator
            case OrchestratorTypes.OPENSHIFT:
                orchestrator = new OpenShift(namespace)
                return orchestrator
        }
    }
}

enum OrchestratorTypes {
    K8S,
    OPENSHIFT,
}


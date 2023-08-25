package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.ServiceAccountServiceGrpc
import io.stackrox.proto.storage.ServiceAccountOuterClass
import objects.K8sServiceAccount
import util.Timer

@Slf4j
class ServiceAccountService extends BaseService {
    static getServiceAccountService() {
        return ServiceAccountServiceGrpc.newBlockingStub(getChannel())
    }

    static getServiceAccounts(RawQuery query = RawQuery.newBuilder().build()) {
        return getServiceAccountService().listServiceAccounts(query).getSaAndRolesList()
    }

    static getServiceAccountDetails(String id) {
        try {
            return getServiceAccountService().getServiceAccount(
                    Common.ResourceByID.newBuilder().setId(id).build()
            ).getSaAndRole()
        } catch (Exception e) {
            log.warn("Error fetching service account", e)
        }
    }

    static RawQuery getServiceAccountQuery(K8sServiceAccount serviceAccount) {
        def query = "Namespace:\"${serviceAccount.namespace}\"+Service Account:\"${serviceAccount.name}\""
        return RawQuery.newBuilder().setQuery(query).build()
    }

    static boolean waitForServiceAccount(K8sServiceAccount serviceAccount) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Service Account"
            def serviceAccounts = getServiceAccounts(getServiceAccountQuery(serviceAccount))
            if (serviceAccounts.size() > 0) {
                return true
            }
        }
        log.warn "Time out for Waiting for Service Account"
        return false
    }

    static boolean waitForServiceAccountRemoved(K8sServiceAccount serviceAccount) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for Service Account removed"
            def serviceAccounts = getServiceAccounts(getServiceAccountQuery(serviceAccount))
            def sa = serviceAccounts.find {
                it.getServiceAccount().name == serviceAccount.name &&
                        it.getServiceAccount().namespace == serviceAccount.namespace
            }
            if (!sa) {
                return true
            }
        }
        log.warn "Time out for Waiting for Service Account removed"
        return false
    }

    @SuppressWarnings(["IfStatementCouldBeTernary", "UnnecessaryIfStatement"])
    static boolean matchServiceAccounts(K8sServiceAccount k8s, ServiceAccountOuterClass.ServiceAccount sr) {
        if (k8s.name != sr.name) {
            return false
        }
        if ((k8s.namespace || sr.namespace) && k8s.namespace != sr.namespace) {
            return false
        }
        if ((k8s.labels || sr.labelsMap) && k8s.labels != sr.labelsMap) {
            return false
        }
        if ((k8s.annotations || sr.annotationsMap) && k8s.annotations != sr.annotationsMap) {
            return false
        }
        if ((k8s.automountToken || sr.automountToken) && k8s.automountToken != sr.automountToken) {
            return false
        }
        if ((k8s.imagePullSecrets || sr.imagePullSecretsList) && k8s.imagePullSecrets != sr.imagePullSecretsList) {
            return false
        }

        return true
    }
}

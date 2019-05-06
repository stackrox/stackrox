package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.ServiceAccountServiceGrpc
import objects.K8sServiceAccount
import util.Timer

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
            println "Error fetching service account: ${e.toString()}"
        }
    }

    static boolean waitForServiceAccount(K8sServiceAccount serviceAccount) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for Service Account"
            def serviceAccounts = getServiceAccounts()
            def sa = serviceAccounts.find {
                it.getServiceAccount().name == serviceAccount.name &&
                    it.getServiceAccount().namespace == serviceAccount.namespace
            }

            if (sa) {
                return true
            }
        }
        println "Time out for Waiting for Service Account"
        return false
    }

    static boolean waitForServiceAccountRemoved(K8sServiceAccount serviceAccount) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for Service Account removed"
            def serviceAccounts = getServiceAccounts()
            def sa = serviceAccounts.find {
                it.getServiceAccount().name == serviceAccount.name &&
                        it.getServiceAccount().namespace == serviceAccount.namespace
            }
            if (!sa) {
                return true
            }
        }
        println "Time out for Waiting for Service Account removed"
        return false
    }
}

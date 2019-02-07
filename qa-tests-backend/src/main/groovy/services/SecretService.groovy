package services

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.SecretServiceGrpc
import io.stackrox.proto.storage.SecretOuterClass
import io.stackrox.proto.storage.SecretOuterClass.ListSecret

class SecretService extends BaseService {

    static getSecretClient() {
        return SecretServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListSecret> getSecrets() {
        return getSecretClient().listSecrets(RawQuery.newBuilder().build()).secretsList
    }

    static waitForSecret(String id, int timeoutSeconds = 10) {
        int intervalSeconds = 1
        def startTime = System.currentTimeMillis()
        for (int waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            if (getSecret(id) != null ) {
                println "SR found secret ${id} within ${(System.currentTimeMillis() - startTime) / 1000}s"
                return true
            }
            println "Retrying in ${intervalSeconds}..."
            sleep(intervalSeconds * 1000)
        }
        println "SR did not detect the secret ${id}"
        return false
    }

    static SecretOuterClass.Secret getSecret(String id) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < 50000 / intervalSeconds; waitTime++) {
            try {
                SecretOuterClass.Secret sec = getSecretClient().getSecret(ResourceByID.newBuilder().setId(id).build())
                return sec
            } catch (Exception e) {
                println "Exception checking for getting the secret ${id}, retrying...:"
                println e.toString()
                sleep(intervalSeconds * 1000)
            }
        }
        println "Failed to add secret ${id} after waiting ${waitTime * intervalSeconds} seconds"
        return null
    }

}

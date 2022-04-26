package services

import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.SecretServiceGrpc
import io.stackrox.proto.api.v1.SecretServiceOuterClass
import io.stackrox.proto.storage.SecretOuterClass
import io.stackrox.proto.storage.SecretOuterClass.ListSecret
import util.Timer

@Slf4j
class SecretService extends BaseService {

    static getSecretClient() {
        return SecretServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListSecret> getSecrets(RawQuery query = RawQuery.newBuilder().build()) {
        return getSecretClient().listSecrets(query).secretsList
    }

    static waitForSecret(String id, int timeoutSeconds = 10) {
        int intervalSeconds = 1
        int retries = timeoutSeconds / intervalSeconds
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            if (getSecret(id) != null ) {
                log.debug "SR found secret ${id} within ${t.SecondsSince()}s"
                return true
            }
            log.debug "Retrying in ${intervalSeconds}..."
        }
        log.warn "SR did not detect the secret ${id}"
        return false
    }

    static SecretOuterClass.Secret getSecret(String id) {
        int intervalSeconds = 1
        int retries = 50 / intervalSeconds
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            try {
                SecretOuterClass.Secret sec = getSecretClient().getSecret(ResourceByID.newBuilder().setId(id).build())
                return sec
            } catch (Exception e) {
                log.debug("Exception checking for getting the secret ${id}, retrying...", e)
            }
        }
        log.warn "Failed to add secret ${id} after waiting ${t.SecondsSince()} seconds"
        return null
    }

    static SecretServiceOuterClass.ListSecretsResponse listSecrets() {
        return getSecretClient().listSecrets()
    }

}

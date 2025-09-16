package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.SecretServiceGrpc
import io.stackrox.proto.api.v1.SecretServiceOuterClass
import io.stackrox.proto.storage.SecretOuterClass
import io.stackrox.proto.storage.SecretOuterClass.ListSecret
import io.stackrox.annotations.Retry

@Slf4j
@CompileStatic
class SecretService extends BaseService {

    static SecretServiceGrpc.SecretServiceBlockingStub getSecretClient() {
        return SecretServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListSecret> getSecrets(RawQuery query = RawQuery.newBuilder().build()) {
        return getSecretClient().listSecrets(query).secretsList
    }

    @Retry(attempts = 10)
    static void waitForSecret(String id) {
        getSecret(id)
    }

    @Retry(attempts = 50)
    static SecretOuterClass.Secret getSecret(String id) {
        return getSecretClient().getSecret(ResourceByID.newBuilder().setId(id).build())
    }

    static SecretServiceOuterClass.ListSecretsResponse listSecrets() {
        return getSecretClient().listSecrets(RawQuery.newBuilder().build())
    }

}

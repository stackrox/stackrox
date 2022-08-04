package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.SignatureIntegrationServiceGrpc
import io.stackrox.proto.storage.SignatureIntegrationOuterClass
import util.Timer

@CompileStatic
@Slf4j
class SignatureIntegrationService extends BaseService {
    static SignatureIntegrationServiceGrpc.SignatureIntegrationServiceBlockingStub getSignatureIntegrationClient() {
        return SignatureIntegrationServiceGrpc.newBlockingStub(getChannel())
    }

    static String createSignatureIntegration(SignatureIntegrationOuterClass.SignatureIntegration integration) {
        SignatureIntegrationOuterClass.SignatureIntegration createdIntegration
        Timer t = new Timer(15, 3)
        while (t.IsValid()) {
            try {
                createdIntegration = getSignatureIntegrationClient().postSignatureIntegration(integration)
                log.debug "Signature integration created: " +
                        "${createdIntegration.getName()}: ${createdIntegration.getId()}"
                break
            } catch (Exception e) {
                log.debug("Unable to create signature integration ${integration.getName()}", e)
            }
        }

        if (!createdIntegration || !createdIntegration.getId()) {
            log.error "Unable to create signature integration"
            return ""
        }

        SignatureIntegrationOuterClass.SignatureIntegration foundIntegration
        t = new Timer(15, 3)
        while (t.IsValid()) {
            try {
                foundIntegration =
                        getSignatureIntegrationClient().getSignatureIntegration(
                                getResourceByID(createdIntegration.getId()))
                if (foundIntegration) {
                    return foundIntegration.getId()
                }
            } catch (Exception e) {
                log.debug("Unable to find the created signature integration ${integration.getName()}", e)
            }
        }
        log.error "Unable to find the created signature integration: ${integration.getName()}"
        return ""
    }

    static Boolean deleteSignatureIntegration(String integrationId) {
        try {
            getSignatureIntegrationClient().deleteSignatureIntegration(getResourceByID(integrationId))
        } catch (Exception e) {
            log.error("Failed to delete signature integration with id ${integrationId}", e)
            return false
        }

        Timer t = new Timer(15, 3)
        while (t.IsValid()) {
            try {
                getSignatureIntegrationClient().getSignatureIntegration(getResourceByID(integrationId))
            } catch (StatusRuntimeException e) {
                if (e.status.code == Status.Code.NOT_FOUND) {
                    log.info "Signature integration deleted: ${integrationId}"
                    return true
                }
                log.debug("error getting signature integration", e)
            }
        }
        return false
    }
}

package services

import com.google.protobuf.Timestamp
import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.CredentialExpiryServiceGrpc
import io.stackrox.proto.api.v1.CredentialExpiryServiceOuterClass.GetCertExpiry

@CompileStatic
class CredentialExpiryService extends BaseService {
    static CredentialExpiryServiceGrpc.CredentialExpiryServiceBlockingStub getCredentialExpiryServiceClient() {
        return CredentialExpiryServiceGrpc.newBlockingStub(getChannel())
    }

    static Timestamp getCentralCertExpiry() {
        return getCredentialExpiryServiceClient().getCertExpiry(
            GetCertExpiry.Request.newBuilder().setComponent(GetCertExpiry.Component.CENTRAL).build()
        ).getExpiry()
    }

    static Timestamp getScannerCertExpiry() {
        return getCredentialExpiryServiceClient().getCertExpiry(
            GetCertExpiry.Request.newBuilder().setComponent(GetCertExpiry.Component.SCANNER).build()
        ).getExpiry()
    }
}

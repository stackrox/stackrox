package services

import com.google.protobuf.Timestamp
import io.stackrox.proto.api.v1.CredentialExpiryServiceGrpc
import io.stackrox.proto.api.v1.CredentialExpiryServiceOuterClass.GetCertExpiry

class CredentialExpiryService extends BaseService {
    static getCredentialExpiryServiceClient() {
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

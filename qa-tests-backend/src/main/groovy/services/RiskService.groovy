package services

import io.stackrox.proto.api.v1.RiskServiceGrpc

class RiskService extends BaseService {
    static getRiskService() {
        return RiskServiceGrpc.newBlockingStub(getChannel())
    }

    static getRisk() {
        return getRiskService().getRisk()
    }
}

package services

import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.api.v1.SummaryServiceGrpc

class SummaryService extends BaseService {
    static getClient() {
        return SummaryServiceGrpc.newBlockingStub(getChannel())
    }

    static getCounts() {
        return getClient().getSummaryCounts(EmptyOuterClass.Empty.newBuilder().build())
    }
}

package services

import stackrox.generated.EmptyOuterClass
import stackrox.generated.SummaryServiceGrpc

class SummaryService extends BaseService {
    static getClient() {
        return SummaryServiceGrpc.newBlockingStub(getChannel())
    }

    static getCounts() {
        return getClient().getSummaryCounts(EmptyOuterClass.Empty.newBuilder().build())
    }
}

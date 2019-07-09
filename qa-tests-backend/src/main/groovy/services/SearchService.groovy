package services

import io.stackrox.proto.api.v1.SearchServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawSearchRequest

class SearchService extends BaseService {
    static getSearchService() {
        return SearchServiceGrpc.newBlockingStub(getChannel())
    }

    static search(RawSearchRequest query) {
        return getSearchService().search(query)
    }

    static autocomplete(RawSearchRequest query) {
        return getSearchService().autocomplete(query)
    }
}

package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.SearchServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawSearchRequest

@CompileStatic
class SearchService extends BaseService {
    static SearchServiceGrpc.SearchServiceBlockingStub getSearchService() {
        return SearchServiceGrpc.newBlockingStub(getChannel())
    }

    static search(RawSearchRequest query) {
        return getSearchService().search(query)
    }

    static autocomplete(RawSearchRequest query) {
        return getSearchService().autocomplete(query)
    }

    static options(SearchServiceOuterClass.SearchCategory category) {
        return getSearchService().options(
                SearchServiceOuterClass.SearchOptionsRequest.
                        newBuilder().
                            addCategories(category).
                            build()
        )
    }
}

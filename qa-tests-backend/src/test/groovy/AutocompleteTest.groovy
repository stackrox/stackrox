import groups.BAT
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawSearchRequest
import io.stackrox.proto.api.v1.SearchServiceOuterClass.SearchCategory
import org.junit.experimental.categories.Category
import services.SearchService

class AutocompleteTest extends BaseSpecification {
    @Category([BAT])
    def "Verify Autocomplete: #query #category #contains"() {
        when:
        SearchServiceOuterClass.AutocompleteResponse resp = SearchService.autocomplete(
                RawSearchRequest.newBuilder()
                        .addAllCategories(category)
                        .setQuery(query)
                        .build()
        )

        then:
        resp.valuesList.contains(contains)

        where:
        "Data inputs are: "
        query                 | category                   | contains
        "Subject:system:auth" | []                         | "system:authenticated"
        "Subject:system:auth" | [SearchCategory.SUBJECTS]  | "system:authenticated"

        "Subject Kind:GROUP"  | []                         | "group"
        "Subject Kind:group"  | []                         | "group"
        "Subject Kind:gr"     | []                         | "group"
    }
}

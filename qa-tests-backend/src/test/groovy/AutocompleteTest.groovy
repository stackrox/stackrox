import groups.BAT
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawSearchRequest
import io.stackrox.proto.api.v1.SearchServiceOuterClass.SearchCategory
import org.junit.experimental.categories.Category
import services.SearchService
import spock.lang.Unroll

class AutocompleteTest extends BaseSpecification {
    private static final SearchCategory VULNERABILITY_SEARCH_CATEGORY =
        isPostgresRun() ?
            SearchCategory.IMAGE_VULNERABILITIES :
            SearchCategory.VULNERABILITIES

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

        "Subject Kind:GROUP"  | []                         | isPostgresRun() ? "GROUP" : "group"
        "Subject Kind:group"  | []                         | "group"
        "Subject Kind:gr"     | []                         | "group"
    }

    @Unroll
    @Category([BAT])
    def "Verify #category search options contains #options"() {
        when:
        def resp = SearchService.options(category)

        then:
        resp.optionsList.containsAll(options)

        where:
        "Data inputs are: "
        category                             | options

        SearchCategory.ALERTS                | ["Deployment", "Policy"]
        SearchCategory.DEPLOYMENTS           | ["Deployment", "Process Name",
                                                "Image Tag", "Dockerfile Instruction Keyword", "CVE", "Component"]
        SearchCategory.IMAGES                | ["Cluster", "Deployment",
                                                "Image Tag", "Dockerfile Instruction Keyword", "CVE", "Component"]
        VULNERABILITY_SEARCH_CATEGORY        | ["Cluster", "Deployment",
                                                "Image Tag", "Dockerfile Instruction Keyword", "CVE", "Component"]
        SearchCategory.IMAGE_COMPONENTS      | ["Cluster", "Deployment",
                                                "Image Tag", "Dockerfile Instruction Keyword", "CVE", "Component"]
        SearchCategory.PODS                  | ["Namespace"]
        SearchCategory.POLICIES              | ["Policy"]
        SearchCategory.SECRETS               | ["Secret"]
        SearchCategory.PROCESS_INDICATORS    | ["Process Name"]
        SearchCategory.CLUSTERS              | ["Cluster"]
        SearchCategory.NAMESPACES            | ["Cluster", "Namespace"]
        SearchCategory.COMPLIANCE            | ["Cluster", "Control", "Deployment", "Namespace", "Node", "Standard"]
        SearchCategory.NODES                 | ["Cluster", "Node"]
        SearchCategory.SERVICE_ACCOUNTS      | ["Cluster", "Service Account"]
        SearchCategory.ROLES                 | ["Cluster", "Role"]
        SearchCategory.ROLEBINDINGS          | ["Cluster", "Role Binding", "Subject"]
        SearchCategory.SUBJECTS              | ["Subject"]
    }

}

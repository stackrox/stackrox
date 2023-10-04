import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawSearchRequest
import io.stackrox.proto.api.v1.SearchServiceOuterClass.SearchCategory

import services.SearchService

import spock.lang.Tag
import spock.lang.Unroll

class AutocompleteTest extends BaseSpecification {
    private static final SearchCategory VULNERABILITY_SEARCH_CATEGORY =
        isPostgresRun() ?
            SearchCategory.IMAGE_VULNERABILITIES :
            SearchCategory.VULNERABILITIES

    private static final String GROUP_AUTOCOMPLETE = isPostgresRun() ? "GROUP" : "group"

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    @Tag("PZ")
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
        "Subject Kind:GROUP"  | []                         | GROUP_AUTOCOMPLETE
        "Subject Kind:group"  | []                         | GROUP_AUTOCOMPLETE
        "Subject Kind:gr"     | []                         | GROUP_AUTOCOMPLETE
    }

    @Unroll
    @Tag("BAT")
    @Tag("COMPATIBILITY")
    @Tag("PZ")
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

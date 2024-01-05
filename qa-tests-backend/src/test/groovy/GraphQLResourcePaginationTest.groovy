
import objects.Pagination
import objects.SortOption
import services.GraphQLService

import org.junit.Assume
import spock.lang.Tag
import spock.lang.Unroll

class GraphQLResourcePaginationTest extends BaseSpecification {

    @Unroll
    @Tag("BAT")
    def "Verify graphql/sublist pagination #topResource #topLevelQuery #topLevelSortOptionDesc #subResource"() {
        given:
        "Ensure on GKE"
        Assume.assumeTrue(orchestrator.isGKE())

        when:
        "Fetch top level query"
        def gqlService = new GraphQLService()
        def query = "query get${topResource}(\$query: String, \$pagination: Pagination) { " +
                "${topResource}s(query: \$query, pagination:\$pagination) { id } }"

        def topLevelPagination = new Pagination(1, 0)
        topLevelPagination.sortOption = topLevelSortOption

        def resultRet = gqlService.Call(query, [ query: topLevelQuery, pagination: topLevelPagination ])
        assert resultRet.hasNoErrors()

        def objs = resultRet.getValue()["${topResource}s"]
        assert objs.size() != 0

        log.info "Got top level objects: ${objs}"

        def sublistGraphQLQuery = "query get${topResource}_${subResource}(" +
                "\$id: ID!, \$query: String, \$pagination: Pagination) {" +
                "${topResource}(id:\$id) { ${subResource}(query: \$query, pagination: \$pagination) { id } } }"
        if (subResource == "") {
            sublistGraphQLQuery = ""
        }
        resultRet = gqlService.Call(sublistGraphQLQuery, [ id: objs[0].id, query: "", pagination: new Pagination(1, 0)])

        then:
        "Validate response code"
        assert sublistGraphQLQuery == "" || resultRet.hasNoErrors()
        assert sublistGraphQLQuery == "" || resultRet.getValue()["${topResource}"]["${subResource}"].size() != 0

        where:
        topResource  | topLevelQuery | topLevelSortOption | topLevelSortOptionDesc | subResource

        "deployment" | "Namespace:stackrox+Deployment:c"       | new SortOption("Deployment", true) \
            | "Sort(Deployment)" | "images"
        "deployment" | "Namespace:stackrox+Deployment:central" | new SortOption("Deployment", true) \
            | "Sort(Deployment)" | "secrets"

        "cluster"    | "" | null | "Sort(null)" | "subjects"
        "cluster"    | "" | null | "Sort(null)" | "serviceAccounts"
        "cluster"    | "" | null | "Sort(null)" | "k8sRoles"

        "node"       | "" | null | "Sort(null)" | ""

        // TODO: re-activate once fixed against postgres
        //"image"      | "Image:main" | null | "Sort(null)" | "deployments"

        "secret"     | "Secret:scanner-db-password" | null | "Sort(null)" | "deployments"

        "subject"    | "Subject:kubelet" | null | "Sort(null)" | "k8sRoles"

        "k8sRole"    | "Role:system:node-bootstrapper" | null | "Sort(null)" | "subjects"
        "k8sRole"    | "Namespace:stackrox+Role:edit"  | null | "Sort(null)" | "serviceAccounts"

        "serviceAccount" | "Service Account:\"central\"" | null | "Sort(null)" | "k8sRoles"
    }

    @Unroll
    @Tag("BAT")
    def "Verify graphql pagination and sublist pagination for namespaces #topLevelQuery #subResource"() {
        given:
        "Check on GKE"
        Assume.assumeTrue(orchestrator.isGKE())

        when:
        "Fetch top level query"
        def gqlService = new GraphQLService()
        def query = "query getNamespaces(\$query: String, \$pagination: Pagination) {" +
                "namespaces(query: \$query, pagination:\$pagination) { metadata { id name } } }"

        def pag = new Pagination(1, 0)

        def resultRet = gqlService.Call(query, [query: topLevelQuery, pagination: pag])
        assert resultRet.hasNoErrors()

        def objs = resultRet.getValue()["namespaces"]
        assert objs.size() != 0

        def sublistGraphQLQuery = "query getNamespace_${subResource}" +
                "(\$id: ID!, \$query: String, \$pagination: Pagination) {" +
                "namespace(id:\$id) { ${subResource}(query: \$query, pagination: \$pagination) { id } } }"

        resultRet = gqlService.Call(sublistGraphQLQuery, [id: objs[0].metadata.id, query: "", pagination: pag])

        then:
        "Validate response code"
        assert sublistGraphQLQuery == "" || resultRet.hasNoErrors()
        assert sublistGraphQLQuery == "" || resultRet.getValue()["namespace"]["${subResource}"].size() != 0

        where:
        topLevelQuery | subResource

        "Namespace:stackrox" | "secrets"
        "Namespace:stackrox" | "serviceAccounts"
        "Namespace:stackrox" | "k8sRoles"
    }

}

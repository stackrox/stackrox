import static Services.getSearchResponse
import static Services.waitForViolation
import objects.Deployment
import spock.lang.Unroll
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import org.junit.experimental.categories.Category
import groups.BAT

class GlobalSearch extends BaseSpecification {

    static final private DEPLOYMENT = new Deployment()
            .setName("qaglobalsearch")
            .setImage("busybox")
            .addPort(22)
            .addLabel("app", "test")
            .setCommand(["sleep", "600"])

    def setupSpec() {
        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)
        // Wait for the latest tag violation since we try to search by it.
        assert waitForViolation(DEPLOYMENT.getName(), "Latest tag")
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(DEPLOYMENT)
    }

    @Unroll
    @Category(BAT)
    def "Verify Global search (#query, #searchCategories)"(
            String query, List<SearchServiceOuterClass.SearchCategory> searchCategories,
            String expectedResultPrefix,
            List<SearchServiceOuterClass.SearchCategory> expectedCategoriesInResult) {

        // This assertion is a validation on the test inputs, to ensure some consistency.
        // If searchCategories are specified in the request, then the expected categories in the result
        // will be exactly the categories specified in searchCategories.
        // We only want to specify expectedCategoriesInResult if we're a search across all categories.
        def expectedCategoriesSet = expectedCategoriesInResult.toSet()
        if (searchCategories.size() > 0) {
            assert expectedCategoriesInResult.empty
            expectedCategoriesSet = searchCategories.toSet()
        }

        when:
        "Run a global search request"
        SearchServiceOuterClass.SearchResponse searchResponse = null
        Set<SearchServiceOuterClass.SearchCategory> presentCategories = null
        withRetry(30, 1) {
            searchResponse = getSearchResponse(query, searchCategories)
            presentCategories = searchResponse.countsList.collectMany {
                count -> count.count > 0 ? [count.category] : [] } .toSet()
            assert presentCategories.size() >= expectedCategoriesSet.size()
        }

        then:
        "Verify that the search response contains what we expect"
        assert !searchResponse?.resultsList?.empty

        // If the test case has an expectedResultPrefix, assert that the result starts with the prefix.
        // Doing a prefix match instead of an exact match because we do prefix search, and if a query happens
        // to match something else.
        if (expectedResultPrefix.size() > 0) {
            searchResponse.resultsList.each {
                assert it.name.startsWith(expectedResultPrefix)
            }
        }

        assert expectedCategoriesSet == presentCategories

        where:
        "Data inputs are :"

        query | searchCategories | expectedResultPrefix | expectedCategoriesInResult

        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS] |
                "qaglobalsearch" | []

        "Image:docker.io/library/busybox:latest" | [SearchServiceOuterClass.SearchCategory.IMAGES] |
                "docker.io/library/busybox:latest" | []

        "Policy:Latest tag" | [SearchServiceOuterClass.SearchCategory.POLICIES] | "Latest tag" | []

        // This implicitly depends on the policy above triggering on the deployment created during this test.
        "Violation State:ACTIVE+Policy:Latest" | [SearchServiceOuterClass.SearchCategory.ALERTS] | "Latest" | []

        // Test passing more than one category.
        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                                       SearchServiceOuterClass.SearchCategory.ALERTS] | "" | []

        // Test options that do not apply to deployments and images, but are global in nature
        "Policy:Latest tag" | [] | "Latest tag" | [SearchServiceOuterClass.SearchCategory.POLICIES,
                                                   SearchServiceOuterClass.SearchCategory.ALERTS]

        // The following two tests make sure that global search gives you all categories
        // when you don't specify a category.
        "Deployment:qaglobalsearch" | [] | "" |
                [SearchServiceOuterClass.SearchCategory.IMAGES, SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                SearchServiceOuterClass.SearchCategory.ALERTS]

        "Image:docker.io/library/busybox:latest" | [] | "" |
                [SearchServiceOuterClass.SearchCategory.IMAGES, SearchServiceOuterClass.SearchCategory.DEPLOYMENTS]
    }

}

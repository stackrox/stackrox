import static Services.getSearchResponse
import static Services.waitForViolation
import stackrox.generated.SearchServiceOuterClass.SearchResponse.Count
import objects.Deployment
import spock.lang.Unroll
import stackrox.generated.SearchServiceOuterClass
import org.junit.experimental.categories.Category
import groups.BAT

class GlobalSearch extends BaseSpecification {

    static final private DEPLOYMENT = new Deployment()
            .setName("qaglobalsearch")
            .setImage("nginx")
            .addPort(22)
            .addLabel("app", "test")

    def setupSpec() {
        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT.getDeploymentUid())
        // Wait for the latest tag violation since we try to search by it.
        assert waitForViolation(DEPLOYMENT.getName(), "Latest tag")
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(DEPLOYMENT.getName())
    }

    @Unroll
    @Category(BAT)
    def "Verify Global search: C961"(String query, List<SearchServiceOuterClass.SearchCategory> searchCategories,
                                     String expectedResultPrefix,
                                     List<SearchServiceOuterClass.SearchCategory> expectedCategoriesInResult) {

        // This assertion is a validation on the test inputs, to ensure some consistency.
        // If searchCategories are specified in the request, then the expected categories in the result
        // will be exactly the categories specified in searchCategories.
        // We only want to specify expectedCategoriesInResult if we're a search across all categories.
        if (searchCategories.size() > 0) {
            assert expectedCategoriesInResult.isEmpty()
        }

        when:
        "Run a global search request"
        def searchResponse = getSearchResponse(query, searchCategories)

        then:
        "Verify that the search response contains what we expect"
        assert searchResponse.resultsList.size() > 0

        // If the test case has an expectedResultPrefix, assert that the result starts with the prefix.
        // Doing a prefix match instead of an exact match because we do prefix search, and if a query happens
        // to match something else.
        if (expectedResultPrefix.size() > 0) {
            assert searchResponse.resultsList.get(0).getName().startsWith(expectedResultPrefix)
        }

        Set<SearchServiceOuterClass.SearchCategory> presentCategories = [] as Set
        for (Count count : searchResponse.getCountsList()) {
            if (count.getCount() > 0) {
                presentCategories.add(count.getCategory())
            }
        }

        // If searchCategories are explicitly specified, we expect results for all of them (and no others!).
        if (searchCategories.size() > 0) {
            assert searchCategories.toSet() == presentCategories
        } else {
            // Else, we just make sure that the expectedCategories passed in all exist in the results.
            assert presentCategories.containsAll(expectedCategoriesInResult)
        }

        where:
        "Data inputs are :"

        query | searchCategories | expectedResultPrefix | expectedCategoriesInResult

        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS] |
                "qaglobalsearch" | []

        "Image:docker.io/library/nginx:latest" | [SearchServiceOuterClass.SearchCategory.IMAGES] |
                "docker.io/library/nginx:latest" | []

        "Policy:Latest tag" | [SearchServiceOuterClass.SearchCategory.POLICIES] | "Latest tag" | []

        // This implicitly depends on the policy above triggering on the deployment created during this test.
        "Violation:Latest" | [SearchServiceOuterClass.SearchCategory.ALERTS] | "Latest" | []

        // Test passing more than one category.
        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                                       SearchServiceOuterClass.SearchCategory.ALERTS] | "" | []

        // The following two tests make sure that global search gives you all categories
        // when you don't specify a category.
        "Deployment:qaglobalsearch" | [] | "" |
                [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS, SearchServiceOuterClass.SearchCategory.ALERTS]

        "Image:docker.io/library/nginx:latest" | [] | "" |
                [SearchServiceOuterClass.SearchCategory.IMAGES, SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                 SearchServiceOuterClass.SearchCategory.ALERTS]
    }

}

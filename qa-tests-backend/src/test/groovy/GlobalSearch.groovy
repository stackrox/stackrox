import static Services.getSearchResponse
import static Services.waitForViolation
import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.SearchServiceOuterClass

import objects.Deployment
import util.Env

import spock.lang.Tag
import spock.lang.Unroll

class GlobalSearch extends BaseSpecification {
    static final private List<SearchServiceOuterClass.SearchCategory> EXPECTED_DEPLOYMENT_CATEGORIES = []
    static final private List<SearchServiceOuterClass.SearchCategory> EXPECTED_IMAGE_CATEGORIES = []

    static final private DEPLOYMENT = new Deployment()
            .setName("qaglobalsearch")
            .setImage("quay.io/rhacs-eng/qa-multi-arch-busybox:latest")
            .addPort(22)
            .addLabel("app", "test")
            .setCommand(["sleep", "600"])

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = 30

    def setupSpec() {
        if (Env.get("ROX_POSTGRES_DATASTORE", null) == "true") {
            EXPECTED_DEPLOYMENT_CATEGORIES.addAll(SearchServiceOuterClass.SearchCategory.CLUSTERS,
                                              SearchServiceOuterClass.SearchCategory.NAMESPACES,
                                              SearchServiceOuterClass.SearchCategory.IMAGES,
                                              SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                                              SearchServiceOuterClass.SearchCategory.ALERTS)
            EXPECTED_IMAGE_CATEGORIES.addAll(SearchServiceOuterClass.SearchCategory.CLUSTERS,
                                         SearchServiceOuterClass.SearchCategory.NAMESPACES,
                                         SearchServiceOuterClass.SearchCategory.IMAGES,
                                         SearchServiceOuterClass.SearchCategory.DEPLOYMENTS)
        } else {
            EXPECTED_DEPLOYMENT_CATEGORIES.addAll(SearchServiceOuterClass.SearchCategory.IMAGES,
                                              SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                                              SearchServiceOuterClass.SearchCategory.ALERTS)
            EXPECTED_IMAGE_CATEGORIES.addAll(SearchServiceOuterClass.SearchCategory.IMAGES,
                                         SearchServiceOuterClass.SearchCategory.DEPLOYMENTS)
        }
        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)
        // Wait for the latest tag violation since we try to search by it.
        def foundViolation = waitForViolation(DEPLOYMENT.getName(), "Latest tag", WAIT_FOR_VIOLATION_TIMEOUT)
        if (!foundViolation) {
            def policy = Services.getPolicyByName("Latest tag")
            log.info "'Latest tag' policy:"
            log.info policy.toString()
        }
        assert foundViolation
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(DEPLOYMENT)
    }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify Global search (no policies)(#query, #searchCategories)"(
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
            searchResponse.countsList.forEach {
                count -> log.info "Category: ${count.category}: ${count.count}"
            }
            presentCategories = searchResponse.countsList.collectMany {
                count -> count.count > 0 ? [count.category] : [] } .toSet()
            assert presentCategories.size() >= expectedCategoriesSet.size()
        }

        then:
        "Verify that the search response contains what we expect"
        assert !searchResponse?.resultsList?.empty
        assert expectedCategoriesSet == presentCategories

        where:
        "Data inputs are :"

        query | searchCategories | expectedResultPrefix | expectedCategoriesInResult

        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS] |
                "qaglobalsearch" | []

        "Image:quay.io/rhacs-eng/qa-multi-arch-busybox:latest" | [SearchServiceOuterClass.SearchCategory.IMAGES] |
                "quay.io/rhacs-eng/qa-multi-arch-busybox:latest" | []

        // This implicitly depends on the policy above triggering on the deployment created during this test.
        "Violation State:ACTIVE+Policy:Latest" | [SearchServiceOuterClass.SearchCategory.ALERTS] | "Latest" | []

        // Test passing more than one category.
        "Deployment:qaglobalsearch" | [SearchServiceOuterClass.SearchCategory.DEPLOYMENTS,
                                       SearchServiceOuterClass.SearchCategory.ALERTS] | "" | []

        // The following two tests make sure that global search gives you all categories
        // The following two tests make sure that global search gives you all categories
        // when you don't specify a category.
        "Deployment:qaglobalsearch" | [] | "" | EXPECTED_DEPLOYMENT_CATEGORIES

        "Image:quay.io/rhacs-eng/qa-multi-arch-busybox:latest" | [] | "" | EXPECTED_IMAGE_CATEGORIES

        "Subject:system:auth" | [SearchServiceOuterClass.SearchCategory.SUBJECTS] | "system:authenticated" | []
    }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify Global search on policies (#query, #searchCategories)"(
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
            searchResponse.countsList.forEach {
                count -> log.info "Category: ${count.category}: ${count.count}"
            }
            presentCategories = searchResponse.countsList.collectMany {
                count -> count.count > 0 ? [count.category] : [] } .toSet()
            assert presentCategories.size() >= expectedCategoriesSet.size()
        }

        then:
        "Verify that the search response contains what we expect"
        assert !searchResponse?.resultsList?.empty
        assert expectedCategoriesSet == presentCategories

        where:
        "Data inputs are :"

        query | searchCategories | expectedResultPrefix | expectedCategoriesInResult

        "Policy:Latest tag" | [SearchServiceOuterClass.SearchCategory.POLICIES] | "Latest tag" | []

        // Test options that do not apply to deployments and images, but are global in nature
        "Policy:Latest tag" | [] | "Latest tag" | [SearchServiceOuterClass.SearchCategory.POLICIES,
                                                   SearchServiceOuterClass.SearchCategory.ALERTS,
                                                   SearchServiceOuterClass.SearchCategory.POLICY_CATEGORIES]
    }

}

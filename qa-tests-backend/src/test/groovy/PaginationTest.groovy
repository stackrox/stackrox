import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.PaginationOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass

import objects.Deployment
import services.AlertService
import services.DeploymentService
import services.ImageService
import services.SecretService

import spock.lang.Tag

@Tag("PZ")
class PaginationTest extends BaseSpecification {
    static final private Map<String, String> SECRETS = [
            "pagination-secret-1" : null,
            "pagination-secret-2" : null,
            "pagination-secret-3" : null,
            "pagination-secret-4" : null,
            "pagination-secret-5" : null,
            "pagination-secret-6" : null,
    ]
    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName("pagination1")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-32")
                    .addLabel("app", "pagination1")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p1", SECRETS[0]),
            new Deployment()
                    .setName("pagination2")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-31")
                    .addLabel("app", "pagination2")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p2", SECRETS[1]),
            new Deployment()
                    .setName("pagination3")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-30")
                    .addLabel("app", "pagination3")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p3", SECRETS[2]),
            new Deployment()
                    .setName("pagination4")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-29")
                    .addLabel("app", "pagination4")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p4", SECRETS[3]),
            new Deployment()
                    .setName("pagination5")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-28")
                    .addLabel("app", "pagination5")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p5", SECRETS[4]),
            new Deployment()
                    .setName("pagination6")
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-27")
                    .addLabel("app", "pagination6")
                    .setCommand(["sleep", "600"])
                    .addSecretName("p6", SECRETS[5]),
    ]

    static final private IMAGE_AND_TAGS_QUERY = "Image:quay.io/rhacs-eng/qa-multi-arch+"+
            "Image Tag:busybox-1-31,busybox-1-32,busybox-1-27,busybox-1-28,busybox-1-29,busybox-1-30"

    def setupSpec() {
        for (String secretName : SECRETS.keySet()) {
            SECRETS.put(secretName, orchestrator.createSecret(secretName))
        }
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
            assert Services.waitForImage(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        for (String secretName : SECRETS.keySet()) {
            orchestrator.deleteSecret(secretName)
        }
    }

    @Tag("BAT")
    def "Verify deployment pagination"() {
        when:
        "Set pagination limit to 3"
        SearchServiceOuterClass.RawQuery query = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(3).setOffset(0))
                .setQuery("Deployment:pagination")
                .build()
        def deployments = DeploymentService.listDeploymentsSearch(query)

        then:
        "verify result set is 3"
        assert deployments.deploymentsCount == 3

        and:
        "Set limit to 10 with offset to 5 on a total count of 10"
        def query2 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(6).setOffset(3))
                .setQuery("Deployment:pagination")
                .build()
        def deployments2 = DeploymentService.listDeploymentsSearch(query2)

        then:
        "Verify result set is 3"
        assert deployments2.deploymentsCount == 3

        and:
        "Get the same violation set in reversed and non-reversed order"
        def query3 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                        .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                                .setField("Deployment")
                                .setReversed(false)))
                .setQuery("Deployment:pagination")
                .build()
        def deployments3 = DeploymentService.listDeploymentsSearch(query3).deploymentsList*.name
        def query4 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                        .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                                .setField("Deployment")
                                .setReversed(true)))
                .setQuery("Deployment:pagination")
                .build()
        def deployments4 = DeploymentService.listDeploymentsSearch(query4).deploymentsList*.name

        then:
        "make sure the results are the same, just reversed"
        assert deployments3 == deployments4.reverse()
    }

    @Tag("BAT")
    def "Verify image pagination"() {
        when:
        "Set pagination limit to 3"
        SearchServiceOuterClass.RawQuery query = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(3).setOffset(0))
                .setQuery(IMAGE_AND_TAGS_QUERY)
                .build()
        def images = ImageService.getImages(query)

        then:
        "verify result set is 3"
        assert images.size() == 3

        and:
        "Set limit to 10 with offset to 5 on a total count of 10"
        def query2 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(6).setOffset(3))
                .setQuery(IMAGE_AND_TAGS_QUERY)
                .build()
        def images2 = ImageService.getImages(query2)

        then:
        "Verify result set is 3"
        assert images2.size() == 3

        and:
        "Get the same violation set in reversed and non-reversed order"
        def query3 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                .setField("Image")
                .setReversed(false)))
                .setQuery(IMAGE_AND_TAGS_QUERY)
                .build()
        def images3 = ImageService.getImages(query3)*.name
        def query4 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                .setField("Image")
                .setReversed(true)))
                .setQuery(IMAGE_AND_TAGS_QUERY)
                .build()
        def images4 = ImageService.getImages(query4)*.name

        then:
        "make sure the results are the same, just reversed"
        assert images3 == images4.reverse()
    }

    @Tag("BAT")
    def "Verify secret pagination"() {
        when:
        "Set pagination limit to 3"
        SearchServiceOuterClass.RawQuery query = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(3).setOffset(0))
                .setQuery("Secret:pagination-secret")
                .build()
        def secrets = SecretService.getSecrets(query)

        then:
        "verify result set is 3"
        assert secrets.size() == 3

        and:
        "Set limit to 10 with offset to 5 on a total count of 10"
        def query2 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder().setLimit(6).setOffset(3))
                .setQuery("Secret:pagination-secret")
                .build()
        def secrets2 = SecretService.getSecrets(query2)

        then:
        "Verify result set is 3"
        assert secrets2.size() == 3

        and:
        "Get the same violation set in reversed and non-reversed order"
        def query3 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                .setField("Secret")
                .setReversed(false)))
                .setQuery("Secret:pagination-secret")
                .build()
        def secrets3 = SecretService.getSecrets(query3)*.name
        def query4 = SearchServiceOuterClass.RawQuery.newBuilder()
                .setPagination(PaginationOuterClass.Pagination.newBuilder()
                .setSortOption(PaginationOuterClass.SortOption.newBuilder()
                .setField("Secret")
                .setReversed(true)))
                .setQuery("Secret:pagination-secret")
                .build()
        def secrets4 = SecretService.getSecrets(query4)*.name

        then:
        "make sure the results are the same, just reversed"
        assert secrets3 == secrets4.reverse()
    }

    @Tag("BAT")
    def "Verify violation pagination"() {
        given:
        "6 violations exist for pagination"
        for (int i = 1; i <= 6; i++) {
            assert Services.waitForViolation("pagination${i}", "No resource requests or limits specified", 30)
        }

        when:
        "Set pagination limit to 3"
        AlertServiceOuterClass.ListAlertsRequest request = AlertServiceOuterClass.ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:No resource requests or limits specified")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setLimit(3)
                        .setOffset(0)
        ).build()
        def alerts = AlertService.getViolations(request)

        then:
        "verify result set is 3"
        assert alerts.size() == 3

        and:
        "Set limit to 10 with offset to 5 on a total count of 10"
        AlertServiceOuterClass.ListAlertsRequest request2 = AlertServiceOuterClass.ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:No resource requests or limits specified")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setLimit(6)
                        .setOffset(3)
        ).build()
        def alerts2 = AlertService.getViolations(request2)

        then:
        "Verify result set is 3"
        assert alerts2.size() == 3

        and:
        "Get the same violation set in reversed and non-reversed order"
        AlertServiceOuterClass.ListAlertsRequest request3 = AlertServiceOuterClass.ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:No resource requests or limits specified")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setSortOption(
                        PaginationOuterClass.SortOption.newBuilder()
                                .setField("Policy")
                                .setReversed(false))
        ).build()
        def alerts3 = AlertService.getViolations(request3).collect { it.policy.name }
        AlertServiceOuterClass.ListAlertsRequest request4 = AlertServiceOuterClass.ListAlertsRequest.newBuilder()
                .setQuery("Deployment:pagination+Policy:No resource requests or limits specified")
                .setPagination(
                PaginationOuterClass.Pagination.newBuilder()
                        .setSortOption(
                        PaginationOuterClass.SortOption.newBuilder()
                                .setField("Policy")
                                .setReversed(true))
        ).build()
        def alerts4 = AlertService.getViolations(request4).collect { it.policy.name }

        then:
        "make sure the results are the same, just reversed"
        assert alerts3 == alerts4.reverse()
    }
}

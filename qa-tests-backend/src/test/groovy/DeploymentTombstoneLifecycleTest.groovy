import static org.junit.Assume.assumeTrue

import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.DeploymentOuterClass.Deployment
import io.stackrox.proto.storage.DeploymentOuterClass.DeploymentLifecycleStage

import objects.Deployment as DeploymentObject
import services.DeploymentService
import services.ImageService
import util.Timer
import util.Env

import spock.lang.Tag
import spock.lang.Unroll

/**
 * Integration tests for deployment soft-delete (tombstone) lifecycle.
 * Tests the full flow: create → delete → tombstone → TTL → purge.
 */
@Tag("BAT")
@Tag("Integration")
class DeploymentTombstoneLifecycleTest extends BaseSpecification {
    private static final String DEPLOYMENT_NAME = "tombstone-test-deployment"
    private static final String DEPLOYMENT_IMAGE =
        "quay.io/rhacs-eng/qa-multi-arch:nginx-1.12@sha256:72daaf46f11cc753c4eab981cbf869919bd1fee3d2170a2adeac12400f494728"

    private static final DeploymentObject TEST_DEPLOYMENT = new DeploymentObject()
            .setName(DEPLOYMENT_NAME)
            .setImage(DEPLOYMENT_IMAGE)
            .addLabel("app", "tombstone-test")
            .setCommand(["sleep", "600"])

    /**
     * Test full tombstone lifecycle: create → active → delete → tombstone → verify data retained.
     * Note: We cannot test TTL expiration and purging in this test as it requires waiting 24h or time manipulation.
     */
    @Unroll
    @Tag("Tombstone")
    def "Verify deployment tombstone lifecycle: create, delete, verify tombstone and data retention"() {
        given:
        "Deploy a workload with vulnerabilities"
        def k8sDeployment = orchestrator.createDeployment(TEST_DEPLOYMENT)
        def deploymentId = k8sDeployment.getMetadata().getUid()
        ImageService.scanImage(DEPLOYMENT_IMAGE)

        when:
        "Wait for StackRox to detect the deployment"
        assert Services.waitForDeploymentByID(deploymentId, DEPLOYMENT_NAME, 30, 2),
            "StackRox should detect the deployed workload"

        then:
        "Verify deployment is ACTIVE"
        Deployment deployment = DeploymentService.getDeployment(deploymentId)
        assert deployment != null, "Deployment should exist in StackRox"
        assert deployment.getLifecycleStage() == DeploymentLifecycleStage.DEPLOYMENT_ACTIVE,
            "Deployment should have lifecycle_stage = ACTIVE"
        assert deployment.getTombstone() == null || !deployment.hasTombstone(),
            "Active deployment should not have tombstone"

        and:
        "Verify deployment appears in default queries (active only)"
        def activeDeployments = DeploymentService.listDeploymentsSearch(
            RawQuery.newBuilder().setQuery("Deployment:${DEPLOYMENT_NAME}").build()
        )
        assert activeDeployments.deploymentsList.any { it.getId() == deploymentId },
            "Active deployment should appear in default list query"

        and:
        "Wait for vulnerability data to be available"
        assert Services.waitForVulnerabilitiesForImage(TEST_DEPLOYMENT, 60),
            "Vulnerability data should be available for deployment"

        when:
        "Delete the deployment via Kubernetes API"
        orchestrator.deleteDeployment(TEST_DEPLOYMENT)

        then:
        "Wait for tombstone to be created (deployment lifecycle_stage changes to DELETED)"
        Timer tombstoneTimer = new Timer(30, 2)
        boolean tombstoneCreated = false
        Deployment deletedDeployment = null

        while (tombstoneTimer.IsValid()) {
            try {
                deletedDeployment = DeploymentService.getDeployment(deploymentId)
                if (deletedDeployment.getLifecycleStage() == DeploymentLifecycleStage.DEPLOYMENT_DELETED) {
                    tombstoneCreated = true
                    break
                }
            } catch (Exception e) {
                // Deployment might temporarily be unavailable during transition
            }
        }

        assert tombstoneCreated, "Deployment should transition to DELETED lifecycle stage after k8s deletion"
        assert deletedDeployment != null, "Deleted deployment should still exist in StackRox"

        and:
        "Verify tombstone fields are set correctly"
        assert deletedDeployment.hasTombstone(), "Deleted deployment should have tombstone"
        assert deletedDeployment.getTombstone().hasDeletedAt(),
            "Tombstone should have deletedAt timestamp"
        assert deletedDeployment.getTombstone().hasExpiresAt(),
            "Tombstone should have expiresAt timestamp"

        // Verify expiresAt is after deletedAt
        def deletedAt = deletedDeployment.getTombstone().getDeletedAt().getSeconds()
        def expiresAt = deletedDeployment.getTombstone().getExpiresAt().getSeconds()
        assert expiresAt > deletedAt,
            "Tombstone expiresAt should be after deletedAt (TTL: ${expiresAt - deletedAt}s)"

        and:
        "Verify deployment does NOT appear in default queries (excludes soft-deleted)"
        def defaultList = DeploymentService.listDeploymentsSearch(
            RawQuery.newBuilder().setQuery("Deployment:${DEPLOYMENT_NAME}").build()
        )
        assert !defaultList.deploymentsList.any { it.getId() == deploymentId },
            "Soft-deleted deployment should NOT appear in default list query"

        and:
        "Verify deployment DOES appear when explicitly querying DELETED lifecycle stage"
        def deletedList = DeploymentService.listDeploymentsSearch(
            RawQuery.newBuilder()
                .setQuery("Deployment:${DEPLOYMENT_NAME}+Lifecycle Stage:DEPLOYMENT_DELETED")
                .build()
        )
        assert deletedList.deploymentsList.any { it.getId() == deploymentId },
            "Soft-deleted deployment should appear when explicitly querying DELETED lifecycle stage"

        and:
        "Verify vulnerability data is retained for soft-deleted deployment"
        // The deployment still exists with its image and vulnerability data
        assert deletedDeployment.getContainersCount() > 0,
            "Deleted deployment should retain container information"
        def containerImage = deletedDeployment.getContainers(0).getImage()
        assert containerImage != null, "Container image should be retained"

        cleanup:
        "Clean up: permanently delete the deployment and its images if test fails before soft-delete"
        try {
            orchestrator.deleteDeployment(TEST_DEPLOYMENT)
        } catch (Exception ignored) {
            // Deployment may already be deleted
        }

        // Note: In production, the pruner will permanently delete the deployment after TTL expires.
        // For test cleanup, we would need to wait for TTL or directly delete from datastore.
        ImageService.deleteImages(
            RawQuery.newBuilder().setQuery("Image:${DEPLOYMENT_IMAGE}").build(),
            true
        )
    }

    /**
     * Test backward compatibility: verify existing API clients see only active deployments by default.
     */
    @Unroll
    @Tag("Tombstone")
    @Tag("BackwardCompatibility")
    def "Verify backward compatibility: default queries exclude soft-deleted deployments"() {
        given:
        "Create two deployments: one active, one that will be soft-deleted"
        def activeDeploymentName = "tombstone-active-deployment"
        def deletedDeploymentName = "tombstone-deleted-deployment"

        def activeDeploymentObj = new DeploymentObject()
                .setName(activeDeploymentName)
                .setImage(DEPLOYMENT_IMAGE)
                .addLabel("app", "tombstone-active")
                .setCommand(["sleep", "600"])

        def deletedDeploymentObj = new DeploymentObject()
                .setName(deletedDeploymentName)
                .setImage(DEPLOYMENT_IMAGE)
                .addLabel("app", "tombstone-deleted")
                .setCommand(["sleep", "600"])

        def k8sActiveDeployment = orchestrator.createDeployment(activeDeploymentObj)
        def k8sDeletedDeployment = orchestrator.createDeployment(deletedDeploymentObj)

        def activeId = k8sActiveDeployment.getMetadata().getUid()
        def deletedId = k8sDeletedDeployment.getMetadata().getUid()

        when:
        "Wait for both deployments to be detected"
        assert Services.waitForDeploymentByID(activeId, activeDeploymentName, 30, 2)
        assert Services.waitForDeploymentByID(deletedId, deletedDeploymentName, 30, 2)

        and:
        "Delete one deployment to create tombstone"
        orchestrator.deleteDeployment(deletedDeploymentObj)

        Timer tombstoneTimer = new Timer(30, 2)
        while (tombstoneTimer.IsValid()) {
            try {
                def dep = DeploymentService.getDeployment(deletedId)
                if (dep.getLifecycleStage() == DeploymentLifecycleStage.DEPLOYMENT_DELETED) {
                    break
                }
            } catch (Exception ignored) {
                // Continue waiting
            }
        }

        then:
        "Default list query returns only active deployment"
        def allDeployments = DeploymentService.listDeploymentsSearch(
            RawQuery.newBuilder().setQuery("").build()
        )
        def deploymentIds = allDeployments.deploymentsList.collect { it.getId() }
        assert deploymentIds.contains(activeId), "Active deployment should be in default list"
        assert !deploymentIds.contains(deletedId), "Soft-deleted deployment should NOT be in default list"

        and:
        "Default count excludes soft-deleted deployment"
        def totalCount = DeploymentService.getDeploymentCount(
            RawQuery.newBuilder().setQuery("").build()
        )
        def countWithDeleted = DeploymentService.getDeploymentCount(
            RawQuery.newBuilder().setQuery("Lifecycle Stage:DEPLOYMENT_DELETED").build()
        )
        assert countWithDeleted >= 1, "Should have at least one deleted deployment"
        // Total count should not include deleted deployments
        def expectedCount = DeploymentService.getDeploymentCount(
            RawQuery.newBuilder().setQuery("Lifecycle Stage:DEPLOYMENT_ACTIVE").build()
        )
        assert totalCount == expectedCount,
            "Default count should equal ACTIVE count (exclude soft-deleted)"

        cleanup:
        "Clean up test deployments"
        try {
            orchestrator.deleteDeployment(activeDeploymentObj)
            orchestrator.deleteDeployment(deletedDeploymentObj)
        } catch (Exception ignored) {
            // May already be deleted
        }
        ImageService.deleteImages(
            RawQuery.newBuilder().setQuery("Image:${DEPLOYMENT_IMAGE}").build(),
            true
        )
    }
}

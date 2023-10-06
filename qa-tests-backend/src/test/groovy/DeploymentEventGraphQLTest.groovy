import static Services.getPolicies
import static Services.waitForViolation
import static util.Helpers.withRetry

import java.util.stream.Collectors

import objects.Deployment
import services.GraphQLService

import spock.lang.Tag
import util.Env

@Tag("BAT")
@Tag("GraphQL")

class DeploymentEventGraphQLTest extends BaseSpecification {
    private static final String DEPLOYMENT_NAME = "eventnginx"
    private static final String PARENT_NAME = "/bin/sh"
    private static final String PROCESS_NAME = "/bin/sleep"
    private static final String PROCESS_ARGS = "600"
    private static final String CONTAINER_NAME = "eventnginx"
    private static final Deployment DEPLOYMENT = new Deployment()
            .setName(DEPLOYMENT_NAME)
            .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-" +
                    "204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad")
            .addLabel("app", "test")
            .setCommand(["sh", "-c", "apt-get -y clean && sleep 600"])
    private static final POLICY = "Ubuntu Package Manager Execution"

    private static final String GET_DEPLOYMENT_EVENTS_OVERVIEW = """
    query getDeploymentEventsOverview(\$deploymentId: ID!) {
        result: deployment(id: \$deploymentId) {
            numPolicyViolations: failingRuntimePolicyCount
            numProcessActivities: processActivityCount
            numRestarts: containerRestartCount
            numTerminations: containerTerminationCount
            numTotalPods: podCount
        }
    }"""

    private static final String GET_POD_EVENTS = """
    query getPodEvents(\$podsQuery: String) {
        result: pods(query: \$podsQuery) {
            id
            name
            containerCount
            events {
                __typename
                id
                name
                timestamp
                ... on ProcessActivityEvent {
                    args
                    uid
                    parentName
                    parentUid
                    inBaseline
                }
            }
        }
    }"""

    private static final String GET_CONTAINER_EVENTS = """
    query getContainerEvents(\$containersQuery: String) {
        result: groupedContainerInstances(query: \$containersQuery) {
            id
            podId
            name
            startTime
            events {
                __typename
                id
                name
                timestamp
                ... on ProcessActivityEvent {
                    args
                    uid
                    parentName
                    parentUid
                    inBaseline
                }
            }
        }
    }"""

    def setupSpec() {
        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(DEPLOYMENT)
    }

    private final gqlService = new GraphQLService()

    def "Verify Deployment Events in GraphQL"() {
        when:
        "Validate Policy Violation is Triggered"

        // Verify this policy exists before waiting for it.
        assert getPolicies().stream()
                .filter { f -> f.getName() == POLICY }
                .collect(Collectors.toList()).size() == 1

        // Wait for the policy violation to be triggered.
        assert waitForViolation(DEPLOYMENT_NAME, POLICY, 66)

        // Wait 30 seconds to ensure all processes start up
        sleep(30_000)

        then:
        "Validate Triggered Deployment Events"

        String deploymentUid = DEPLOYMENT.deploymentUid
        assert deploymentUid != null
        verifyDeploymentEvents(deploymentUid)
        String podUid = verifyPodEvents(deploymentUid)
        assert podUid != null
        verifyContainerEvents(podUid)
    }

    private void verifyDeploymentEvents(String deploymentUid, int retries = 30, int interval = 4) {
        withRetry(retries, interval) {
            def depEvents = gqlService.Call(GET_DEPLOYMENT_EVENTS_OVERVIEW, [deploymentId: deploymentUid])
            assert depEvents.getCode() == 200
            log.info "return code " + depEvents.getCode()
            assert depEvents.getValue().result != null
            def events = depEvents.getValue().result
            if (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x") {
                // observing more than 1 policy voilations randomly
                assert events.numPolicyViolations >= 1
            } else {
                assert events.numPolicyViolations == 1
            }
            // Cannot determine how many processes will actually run at this point due to the apt-get.
            // As long as we see more than 1, we'll take it.
            assert events.numProcessActivities > 1
            assert events.numRestarts == 0
            assert events.numTerminations == 0
            assert events.numTotalPods == 1
        }
    }

    private String verifyPodEvents(String deploymentUid, int retries = 30, int interval = 4) {
        def event = null
        withRetry(retries, interval) {
            def podEvents = gqlService.Call(GET_POD_EVENTS, [podsQuery: "Deployment ID: " + deploymentUid])
            assert podEvents.getCode() == 200
            log.info "return code " + podEvents.getCode()
            assert podEvents.getValue().result != null
            assert podEvents.getValue().result.size() == 1
            event = podEvents.getValue().result.get(0)
            def pod = DEPLOYMENT.getPods().get(0)
            assert event.name == pod.name
            // No need to test start time, as it is tested in the non-groovy API tests.
            assert event.containerCount == 1
            def procEvent = event.events.find { it.name == PROCESS_NAME }
            assert procEvent.parentName == PARENT_NAME
            assert procEvent.parentUid == 0
            assert procEvent.args == PROCESS_ARGS
            assert procEvent.inBaseline
        }
        return event?.id
    }

    private void verifyContainerEvents(String podUid, int retries = 30, int interval = 4) {
        withRetry(retries, interval) {
            def containerEvents = gqlService.Call(GET_CONTAINER_EVENTS, [containersQuery: "Pod ID: " + podUid])
            assert containerEvents.getCode() == 200
            log.info "return code " + containerEvents.getCode()
            assert containerEvents.getValue().result != null
            assert containerEvents.getValue().result.size() == 1
            def event = containerEvents.getValue().result.get(0)
            assert event.name == CONTAINER_NAME
            def procEvent = event.events.find { it.name == PROCESS_NAME }
            assert procEvent.parentName == PARENT_NAME
            assert procEvent.parentUid == 0
            assert procEvent.args == PROCESS_ARGS
            assert procEvent.inBaseline
        }
    }
}

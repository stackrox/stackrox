import static Services.waitForViolation
import static Services.waitForSuspiciousProcessInRiskIndicators
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.storage.AlertOuterClass
import services.AlertService

import common.Constants
import groups.BAT

import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import objects.Deployment
import org.apache.commons.lang.StringUtils
import org.junit.Assume

import org.junit.experimental.categories.Category

import services.ProcessWhitelistService
import spock.lang.Unroll

class ProcessWhiteListsTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX = "deploymentnginx"
    static final private String DEPLOYMENTNGINX_RESOLVE_VIOLATION = "deploymentnginx-violation-resolve"
    static final private String DEPLOYMENTNGINX_RESOLVE_AND_WHITELIST_VIOLATION =
            "deploymentnginx-violation-resolve-whitelist"
    static final private String DEPLOYMENTNGINX_SOFTLOCK = "deploymentnginx-softlock"

    static final private List<Deployment> DEPLOYMENTS =
            [
                    new Deployment()
                     .setName(DEPLOYMENTNGINX)
                     .setImage("nginx:1.7.9")
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_RESOLVE_VIOLATION)
                     .setImage("nginx:1.7.9")
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_RESOLVE_AND_WHITELIST_VIOLATION)
                     .setImage("nginx:1.7.9")
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_SOFTLOCK)
                     .setImage("nginx:1.7.9")
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
            ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }

        //need to  delete whitelists for the container deployed after each test
    }

    @Unroll
    @Category(BAT)
    def "Verify  whitelist processes for the given key before and after locking "() {
        Assume.assumeTrue(Constants.RUN_PROCESS_WHITELIST_TESTS)
        when:
        def deploymentId = DEPLOYMENTS.find { it.name == deploymentName }.getDeploymentUid()
        // Currently, we always create a deployment where the container name is the same
        // as the deployment name
        def containerName = deploymentName
        "get process whitelists is called for a key"
        ProcessWhitelistOuterClass.ProcessWhitelist whitelist = ProcessWhitelistService.
                getProcessWhitelist(deploymentId, containerName)

        assert (whitelist != null)

        then:
        "Verify  whitelisted processes for a given key before and after calling lock whitelists"
        assert ((whitelist.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (whitelist.key.containerName.equalsIgnoreCase(containerName)))
        assert  whitelist.getElements(0).element.processName.contains(processName)

        //lock the whitelist with the key of the container just deployed
        List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists = ProcessWhitelistService.
                lockProcessWhitelists(deploymentId, containerName, true)
        assert  lockProcessWhitelists.size() == 1
        assert  lockProcessWhitelists.get(0).getElementsList().
            find { it.element.processName.equalsIgnoreCase(processName) } != null

        where:
        "Data inputs are :"
        deploymentName     | processName

        DEPLOYMENTNGINX    | "nginx"
    }

    @Unroll
    @Category(BAT)
    def "Verify whitelist processes violations or no violations after resolving,whitelisting for the given key"() {
               /*
                    a)Lock the whitelists for the key
                    b)exec into the container and run a process
                    c)verify violation alert for Unauthorized Process Execution
                    d)
                        test case :choose to only resolve violation
                            exec into the container and run the  process again and verify violation alert
                        test case : choose to both resolve and whitelist
                            exec into the container and run the  process again and verify no violation alert
               */
        Assume.assumeTrue(Constants.RUN_PROCESS_WHITELIST_TESTS)
        when:
        "get process whitelists is called for a key"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        String deploymentId = deployment.getDeploymentUid()
        String containerName = deployment.getName()
        ProcessWhitelistOuterClass.ProcessWhitelist whitelist = ProcessWhitelistService.
                 getProcessWhitelist(deploymentId, containerName)
        assert ((whitelist.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                 (whitelist.key.containerName.equalsIgnoreCase(containerName)))
        assert whitelist.getElements(0).element.processName.contains(processName)

        List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists = ProcessWhitelistService.
                 lockProcessWhitelists(deploymentId, containerName, true)
        assert (!StringUtils.isEmpty(lockProcessWhitelists.get(0).getElements(0).getElement().processName))
        orchestrator.execInContainer(deployment, "pwd")

        //check for whitelist  violation
        assert waitForViolation(containerName, "Unauthorized Process Execution", 90)
        List<AlertOuterClass.ListAlert> alertList = AlertService.getViolations(AlertServiceOuterClass.ListAlertsRequest
                 .newBuilder().build())
        String alertId
        for (AlertOuterClass.ListAlert alert : alertList) {
            if (alert.getPolicy().name.equalsIgnoreCase("Unauthorized Process Execution") &&
                     alert.deployment.id.equalsIgnoreCase(deploymentId) && resolveWhitelist) {
                alertId = alert.id
                AlertService.resolveAlert(alertId, true)
            }
            else
            {
                alertId = alert.id
                AlertService.resolveAlert(alertId, false)
            }
         }
        orchestrator.execInContainer(deployment, "pwd")
        if (resolveWhitelist) {
            waitForViolation(containerName, "Unauthorized Process Execution", 90)
        }
        else {
            assert waitForViolation(containerName, "Unauthorized Process Execution", 90)
        }
        then:
        "Verify for violation after removing the process from whitelists"
        List<AlertOuterClass.ListAlert> alertListAnother = AlertService
                 .getViolations(AlertServiceOuterClass.ListAlertsRequest
                 .newBuilder().build())
        int numAlertsAfterResolve
        for (AlertOuterClass.ListAlert alert : alertListAnother) {
            if (alert.getPolicy().name.equalsIgnoreCase("Unauthorized Process Execution")
                     && alert.deployment.id.equalsIgnoreCase(deploymentId)) {
                numAlertsAfterResolve++
             }
         }
        System.out.println("numAlertsAfterResolve .. " + numAlertsAfterResolve)
        assert (numAlertsAfterResolve  == expectedViolationsCount)

        where:
        "Data inputs are :"
        deploymentName                                   | processName  | resolveWhitelist | expectedViolationsCount

        DEPLOYMENTNGINX_RESOLVE_VIOLATION               | "nginx"      | false            | 1

        DEPLOYMENTNGINX_RESOLVE_AND_WHITELIST_VIOLATION | "nginx"      | true             | 0
     }
    @Unroll
    @Category(BAT)
    def "Verify  processes risk indicators for the given key after soft-lock "() {
        Assume.assumeTrue(false)
        when:
        "get process whitelists is called for a key"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        String deploymentId = deployment.getDeploymentUid()
        String containerName = deployment.getName()
        ProcessWhitelistOuterClass.ProcessWhitelist whitelist = ProcessWhitelistService.
                    getProcessWhitelist(deploymentId, containerName)
        assert ((whitelist.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (whitelist.key.containerName.equalsIgnoreCase(containerName)))
        assert whitelist.getElements(0).element.processName.contains(processName)
        Thread.sleep(60000)
        orchestrator.execInContainer(deployment, "pwd")
        then:
        DeploymentOuterClass.Risk.Result result = waitForSuspiciousProcessInRiskIndicators(deploymentId, 60)
        assert (result != null)
        DeploymentOuterClass.Risk.Result.Factor factor =  result.factorsList.find { it.message.contains("pwd") }
        assert factor != null
        where:
        "Data inputs are :"
        deploymentName                                   | processName
        DEPLOYMENTNGINX_SOFTLOCK            |   "nginx"
    }
    }

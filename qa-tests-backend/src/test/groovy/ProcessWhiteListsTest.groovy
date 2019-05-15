
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
    static final private String DEPLOYMENTNGINX_DELETE = "deploymentnginx-delete"

    static final private String DEPLOYMENTNGINX_REMOVEPROCESS = "deploymentnginx-removeprocess"
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
             new Deployment()
                     .setName(DEPLOYMENTNGINX_DELETE)
                     .setImage("nginx:1.7.9")
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
                    new Deployment()
                          .setName(DEPLOYMENTNGINX_REMOVEPROCESS)
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

        DEPLOYMENTNGINX    | "/usr/sbin/nginx"
    }
    @Unroll
    @Category(BAT)
    def "Verify whitelist process violation after resolve whitelist "() {
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
        "exec into the container after locking whitelists and create a whitelist violation"
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
        "Verify for violation or no violation after resolve/resolve and whitelist"
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

        DEPLOYMENTNGINX_RESOLVE_VIOLATION               | "/usr/sbin/nginx"      | false            | 1

        DEPLOYMENTNGINX_RESOLVE_AND_WHITELIST_VIOLATION | "/usr/sbin/nginx"      | true             | 0
     }
    @Unroll
    @Category(BAT)
    def "Verify  processes risk indicators for the given key after soft-lock "() {
        when:
        "exec into the container and run a process and wait for soft lock to kick in"
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
        "verify for suspicious process in risk indicator"
        DeploymentOuterClass.Risk.Result result = waitForSuspiciousProcessInRiskIndicators(deploymentId, 60)
        assert (result != null)
        DeploymentOuterClass.Risk.Result.Factor factor =  result.factorsList.find { it.message.contains("pwd") }
        assert factor != null
        where:
        "Data inputs are :"
        deploymentName                      |   processName
        DEPLOYMENTNGINX_SOFTLOCK            |   "/usr/sbin/nginx"
    }

    @Unroll
    @Category(BAT)
    def "Verify whitelists are deleted when their deployment is deleted"() {
        /*
                a)get all whitelists
                b)verify whitelists exist for a deployment
                c)delete the deployment
                d)get all whitelists
                e)verify all whitelists for the deployment have been deleted
        */
        Assume.assumeTrue(Constants.RUN_PROCESS_WHITELIST_TESTS)
        when:
        "a deployment is deleted"
        //Get all whitelists for our deployment and assert they exist
        def deployment = DEPLOYMENTS.find { it.name == DEPLOYMENTNGINX_DELETE }
        assert deployment != null
        String deploymentId = deployment.getDeploymentUid()
        def whitelistsCreated = ProcessWhitelistService.waitForDeploymentWhitelistsCreated(deploymentId)
        assert(whitelistsCreated)

        //Delete the deployment
        orchestrator.deleteDeployment(deployment)
        Services.waitForSRDeletion(deployment)

        then:
        "Verify that all whitelists with that deployment ID have been deleted"
        def whitelistsDeleted = ProcessWhitelistService.waitForDeploymentWhitelistsDeleted(deploymentId)
        assert(whitelistsDeleted)
    }

    @Unroll
    @Category(BAT)
    def "Verify  removed whitelist process not getting added back to whitelist after rerun "() {
        /*
                1.run a process and verify if it exists in the whitelist
                2.remove the process
                3.rerun the process to verify it it does not get added to the whitelist
         */
        Assume.assumeTrue(Constants.RUN_PROCESS_WHITELIST_TESTS)
        when:
        "an added process is removed and whitelist is locked and the process is run"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        def deploymentId = deployment.deploymentUid
        def containerName = deploymentName

        //Wait for whitelist to be created
        def initialWhitelist = ProcessWhitelistService.getProcessWhitelist(deploymentId, containerName)
        assert (initialWhitelist != null)

        //Add the process to the whitelist
        ProcessWhitelistOuterClass.ProcessWhitelistKey [] keys = [
                new ProcessWhitelistOuterClass
                .ProcessWhitelistKey().newBuilderForType().setContainerName(containerName)
                .setDeploymentId(deploymentId).build(),
        ]
        String [] toBeAddedProcesses = ["pwd"]
        String [] toBeRemovedProcesses = []
        List<ProcessWhitelistOuterClass.ProcessWhitelist> updatedList = ProcessWhitelistService
                .updateProcessWhitelists(keys, toBeAddedProcesses, toBeRemovedProcesses)
        assert ( updatedList!= null)
        ProcessWhitelistOuterClass.ProcessWhitelist whitelist = ProcessWhitelistService.
                getProcessWhitelist(deploymentId, containerName)
        List<ProcessWhitelistOuterClass.WhitelistElement> elements = whitelist.elementsList
        ProcessWhitelistOuterClass.WhitelistElement element = elements.find { it.element.processName.contains("pwd") }
        assert ( element != null)

        //Remove the process from the whitelist
        toBeAddedProcesses = []
        toBeRemovedProcesses = ["pwd"]
        List<ProcessWhitelistOuterClass.ProcessWhitelist> updatedListAfterRemoveProcess = ProcessWhitelistService
                .updateProcessWhitelists(keys, toBeAddedProcesses, toBeRemovedProcesses)
        assert ( updatedListAfterRemoveProcess!= null)
        orchestrator.execInContainer(deployment, "pwd")
        then:
        "verify process is not added to the whitelist"
        ProcessWhitelistOuterClass.ProcessWhitelist whitelistAfterReRun = ProcessWhitelistService.
                getProcessWhitelist(deploymentId, containerName)
        assert  ( whitelistAfterReRun.elementsList.find { it.element.processName.contains("pwd") } == null)
        where:
        deploymentName                                   | processName
        DEPLOYMENTNGINX_REMOVEPROCESS           |   "nginx"
    }

    }

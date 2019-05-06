import common.Constants
import groups.BAT
import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import objects.Deployment
import org.junit.Assume

import org.junit.experimental.categories.Category
import services.ProcessWhitelistService
import spock.lang.Unroll

class ProcessWhiteListsE2ETest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX = "deploymentnginx-qatest"

    static final private List<Deployment> DEPLOYMENTS =
           [ new Deployment()
                    .setName(DEPLOYMENTNGINX)
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
        "get process whitelists is called for a key"
        ProcessWhitelistOuterClass.ProcessWhitelist whitelist = ProcessWhitelistService.
                getProcessWhitelist(deploymentId, container_name)

        assert (whitelist != null)

        then:
        "Verify  whitelisted processes for a given key before and after calling lock whitelists"
        assert ((whitelist.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (whitelist.key.containerName.equalsIgnoreCase(container_name)))
        assert  whitelist.getElements(0).element.processName.equalsIgnoreCase(process_name)

        //lock the whitelist with the key of the container just deployed
        List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists = ProcessWhitelistService.
                lockProcessWhitelists(deploymentId, container_name)
        assert  lockProcessWhitelists.size() == 1
        assert  lockProcessWhitelists.get(0).getElementsList().get(0).
                element.processName.equalsIgnoreCase(process_name)

        where:
        "Data inputs are :"
        container_name                 | deploymentId | process_name

        DEPLOYMENTNGINX                 | DEPLOYMENTS[0].deploymentUid| "/usr/sbin/nginx"
    }
}

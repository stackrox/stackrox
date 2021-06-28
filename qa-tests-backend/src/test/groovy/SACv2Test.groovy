import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_WRITE_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.Rules
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.newBuilder
import static services.ApiTokenService.generateToken

import services.BaseService
import services.FeatureFlagService
import services.RoleService

import groups.BAT
import org.junit.experimental.categories.Category
import spock.lang.Requires
import spock.lang.Shared

import io.stackrox.proto.api.v1.ApiTokenService
import io.stackrox.proto.storage.RoleOuterClass

@Category(BAT)
@Requires({ FeatureFlagService.isFeatureFlagEnabled('ROX_SCOPED_ACCESS_CONTROL_V2') })
class SACv2Test extends SACTest {

    @Shared
    private Map<String, RoleOuterClass.Access> allResourcesAccess

    @Shared
    private Map<String, List<String>> tokenToRoles

    def setupSpec() {
        disableAuthzPlugin()

        allResourcesAccess = RoleService.resources.resourcesList.collectEntries { [it, READ_WRITE_ACCESS] }

        def noaccessScope = RoleService.createAccessScope(newBuilder()
                .setName("no-access-scope").build())
        def remoteQaTest1 = createAccessScope("remote", "qa-test1")

        def noaccess = createRole(noaccessScope.id, allResourcesAccess)

        tokenToRoles = [
                (NOACCESSTOKEN)           : [noaccess],
                (ALLACCESSTOKEN)          : [createRole("", allResourcesAccess)],
                "deployments-access-token": [createRole(createAccessScope(
                        "remote", "qa-test2").id, ["Deployment": READ_ACCESS])],
                "getSummaryCountsToken"   : [createRole(remoteQaTest1.id, allResourcesAccess)],
                "listSecretsToken"        : [createRole("", ["Secret": READ_ACCESS])],
                "searchDeploymentsToken"  : [createRole(remoteQaTest1.id, ["Deployment": READ_ACCESS]), noaccess],
                "searchImagesToken"       : [createRole(remoteQaTest1.id, ["Image": READ_ACCESS]), noaccess],
                "searchNamespacesToken"   : [createRole(remoteQaTest1.id, ["Namespace": READ_ACCESS]), noaccess],
                "searchAlertsToken"       : [createRole(remoteQaTest1.id, ["Alert": READ_ACCESS]), noaccess],
                "stackroxNetFlowsToken"   : [createRole(createAccessScope("remote", "stackrox").id,
                        ["Deployment": READ_ACCESS, "NetworkGraph": READ_ACCESS]),
                                             createRole("", ["Cluster": READ_ACCESS]), noaccess],
                "kubeSystemImagesToken"   : [createRole(createAccessScope(
                        "remote", "kube-system").id, ["Image": READ_ACCESS]), noaccess],
        ]
    }

    def cleanupSpec() {
        cleanupRole(*(tokenToRoles.values().flatten().unique()))
    }

    @Override
    def useToken(String tokenName) {
        ApiTokenService.GenerateTokenResponse token = generateToken(tokenName, *(tokenToRoles.get(tokenName)))
        BaseService.useApiToken(token.token)
    }

    @Override
    Boolean summaryTestShouldSeeNoClustersAndNodes() { false }

    def cleanupRole(String... roleName) {
        roleName.each {
            try {
                def role = RoleService.getRole(it)
                RoleService.deleteRole(role.name)
                RoleService.deleteAccessScope(role.accessScopeId)
            } catch (Exception e) {
                println "Error deleting role ${name}: ${e}"
            }
        }
    }

    String createRole(String sacId, Map<String, RoleOuterClass.Access> resources) {
        String id = UUID.randomUUID()
        RoleService.createRole(RoleOuterClass.Role.newBuilder()
                .setId(id)
                .setName("SACv2 Test Automation Role " + id)
                .putAllResourceToAccess(resources)
                .setAccessScopeId(sacId)
                .build()
        ).name
    }

    def createAccessScope(String clusterName, String namespaceName) {
        RoleService.createAccessScope(newBuilder()
                .setName(UUID.randomUUID().toString())
                .setRules(Rules.newBuilder()
                        .addIncludedNamespaces(Rules.Namespace.newBuilder()
                                .setClusterName(clusterName)
                                .setNamespaceName(namespaceName)))
                .build())
    }

}

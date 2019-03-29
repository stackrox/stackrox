import groups.BAT
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.AuthproviderService
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ApiTokenService
import services.AuthProviderService
import services.BaseService
import services.ClusterService
import services.NetworkPolicyService
import services.RoleService
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.storage.RoleOuterClass
import spock.lang.Shared
import spock.lang.Unroll
import util.Env

class RbacAuthTest extends BaseSpecification {
    // Create a map of resource name -> functions to execute in read or write scenarios
    // To add tests for a specific resource, simply add the resource name and functions to the map
    //
    // TESTING NOTE: if you specify permissions for a resource in the BAT test, then you should make sure to have
    // this map filled in for that resource to ensure proper permission testing
    static final private Map<String, Map<RoleOuterClass.Access, Closure>> RESOURCE_FUNCTION_MAP = [
            "Cluster": [(RoleOuterClass.Access.READ_ACCESS):
                                ( { ClusterService.getClusterId() }),
                        (RoleOuterClass.Access.READ_WRITE_ACCESS):
                                ( {
                                    ClusterService.createCluster(
                                        "automation",
                                        "stackrox/main:latest",
                                        "central.stackrox:443")
                                })],
            "NetworkPolicy": [(RoleOuterClass.Access.READ_ACCESS):
                                      ( { NetworkPolicyService.generateNetworkPolicies() }),
                              (RoleOuterClass.Access.READ_WRITE_ACCESS):
                                      ( {
                                          NetworkPolicyService.applyGeneratedNetworkPolicy(
                                                  NetworkPolicyService.generateNetworkPolicies())
                                      })],
    ]

    @Shared
    private String basicAuthServiceId

    def setupSpec() {
        AuthproviderService.GetAuthProvidersResponse providers = AuthProviderService.getAuthProviders()
        basicAuthServiceId = providers.authProvidersList.find { it.type == "basic" }?.id
    }

    def cleanupSpec() {
    }

    def hasReadAccess(String res, Map<String, RoleOuterClass.Access> resource) {
        return resource.get(res) >= RoleOuterClass.Access.READ_ACCESS
    }

    def hasWriteAccess(String res, Map<String, RoleOuterClass.Access> resource) {
        return resource.get(res) == RoleOuterClass.Access.READ_WRITE_ACCESS
    }

    def canDo(Closure closure, String token) {
        BaseService.useApiToken(token)
        def result = closure()
        BaseService.useBasicAuth()
        return !(result instanceof StatusRuntimeException)
    }

    @Unroll
    @Category(BAT)
    def "Verify RBAC with Role/Token combinations: #resourceAccess"() {
        //skip on OS for now
        def skip = false
        if (resourceTest.contains("NetworkPolicy")) {
            skip = Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT
            Assume.assumeFalse(skip)
        }

        when:
        "Create a test role"
        def testRole = RoleOuterClass.Role.newBuilder()
                .setName("Automation Role")
                .putAllResourceToAccess(resourceAccess)
                .build()
        RoleService.createRole(testRole)
        assert RoleService.getRole(testRole.name)
        println "Created Role:\n${testRole}"

        and:
        "Create test API token in that role"
        GenerateTokenResponse token = ApiTokenService.generateToken("Test Token", testRole.name)
        assert token.token != null

        then:
        "verify RBAC permissions"
        for (String resource : resourceTest) {
            def readFunction = RESOURCE_FUNCTION_MAP.get(resource).get(RoleOuterClass.Access.READ_ACCESS)
            def writeFunction = RESOURCE_FUNCTION_MAP.get(resource).get(RoleOuterClass.Access.READ_WRITE_ACCESS)
            println "Testing read function for ${resource}"
            def read = hasReadAccess(resource, resourceAccess) == canDo(readFunction, token.token)
            assert read
            println "Testing write function for ${resource}"
            def write = hasWriteAccess(resource, resourceAccess) == canDo(writeFunction, token.token)
            assert write
        }

        cleanup:
        "remove role and token"
        if (resourceAccess.containsKey("NetworkPolicy") &&
                resourceAccess.get("NetworkPolicy") == RoleOuterClass.Access.READ_WRITE_ACCESS &&
                !skip) {
            NetworkPolicyService.applyGeneratedNetworkPolicy(
                    NetworkPolicyService.undoGeneratedNetworkPolicy().undoModification
            )
        }
        if (testRole?.name != null) {
            RoleService.deleteRole(testRole.name)
        }
        if (token?.metadata?.id != null) {
            ApiTokenService.revokeToken(token.metadata.id)
        }
        def testClusterId = ClusterService.getClusterId("automation")
        if (testClusterId != null) {
            ClusterService.deleteCluster(testClusterId)
        }

        where:
        "Data inputs"

        resourceAccess                                              | resourceTest
        [:]                                                         | ["Cluster"]
        ["Cluster": RoleOuterClass.Access.NO_ACCESS]                | ["Cluster"]
        ["Cluster": RoleOuterClass.Access.READ_ACCESS]              | ["Cluster"]
        ["Cluster": RoleOuterClass.Access.READ_WRITE_ACCESS]        | ["Cluster"]
        ["Cluster": RoleOuterClass.Access.READ_ACCESS,
         "NetworkGraph": RoleOuterClass.Access.READ_ACCESS,
         "NetworkPolicy": RoleOuterClass.Access.READ_ACCESS,]       | ["NetworkPolicy"]
        ["Cluster": RoleOuterClass.Access.READ_ACCESS,
         "NetworkGraph": RoleOuterClass.Access.READ_ACCESS,
         "NetworkPolicy": RoleOuterClass.Access.READ_WRITE_ACCESS,] | ["NetworkPolicy"]
    }
}

import groups.BAT
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.api.v1.AuthproviderService
import io.stackrox.proto.storage.NetworkPolicyOuterClass
import io.stackrox.proto.storage.RoleOuterClass
import org.junit.experimental.categories.Category
import services.ApiTokenService
import services.AuthProviderService
import services.BaseService
import services.ClusterService
import services.NetworkPolicyService
import services.ProcessService
import services.RoleService
import spock.lang.Shared
import spock.lang.Unroll

class RbacAuthTest extends BaseSpecification {

    private static final NETPOL_YAML = """
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: qa-rbac-test-apply
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: qa-rbac-test-apply
  ingress:
  - from: []
"""

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
                                          def netPolMod =
                                                  new NetworkPolicyOuterClass.NetworkPolicyModification.Builder()
                                                      .setApplyYaml(NETPOL_YAML)
                                                      .build()
                                          NetworkPolicyService.applyGeneratedNetworkPolicy(netPolMod)
                                      })],
    ]

    @Shared
    private String basicAuthServiceId

    def setupSpec() {
        BaseService.useBasicAuth()
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

    def canDo(Closure closure, String token, boolean allowOtherError = false) {
        BaseService.setUseClientCert(false)
        BaseService.useApiToken(token)

        try {
            def result = closure()
            if (result instanceof Throwable) {
                throw (Throwable)result
            }
        } catch (StatusRuntimeException ex) {
            if (ex.status.code == Status.Code.PERMISSION_DENIED) {
                return false
            }
            if (!allowOtherError) {
                throw ex
            }
        } finally {
            resetAuth()
        }
        return true
    }

    def myPermissions(String token) {
        BaseService.setUseClientCert(false)
        BaseService.useApiToken(token)

        try {
            return RoleService.myPermissions()
        } finally {
            resetAuth()
        }
    }

    @Unroll
    @Category(BAT)
    def "Verify RBAC with Role/Token combinations: #resourceAccess"() {
        when:
        "Create a test role"
        def testRole = RoleService.createRoleWithScopeAndPermissionSet("Automation Role",
            UNRESTRICTED_SCOPE_ID, resourceAccess)
        assert RoleService.getRole(testRole.name)
        log.info "Created Role:\n${testRole}"

        and:
        "Create test API token in that role"
        GenerateTokenResponse token = ApiTokenService.generateToken("Test Token", testRole.name)
        assert token.token != null

        then:
        "verify RBAC permissions"
        for (String resource : resourceTest) {
            def readFunction = RESOURCE_FUNCTION_MAP.get(resource).get(RoleOuterClass.Access.READ_ACCESS)
            def writeFunction = RESOURCE_FUNCTION_MAP.get(resource).get(RoleOuterClass.Access.READ_WRITE_ACCESS)
            log.info "Testing read function for ${resource}"
            def read = hasReadAccess(resource, resourceAccess) == canDo(readFunction, token.token)
            assert read
            log.info "Testing write function for ${resource}"
            def write = hasWriteAccess(resource, resourceAccess) == canDo(writeFunction, token.token)
            assert write
        }

        cleanup:
        resetAuth()

        "remove role and token"
        if (resourceAccess.containsKey("NetworkPolicy") &&
                resourceAccess.get("NetworkPolicy") == RoleOuterClass.Access.READ_WRITE_ACCESS) {
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
         "Deployment": RoleOuterClass.Access.READ_ACCESS,
         "NetworkGraph": RoleOuterClass.Access.READ_ACCESS,
         "NetworkPolicy": RoleOuterClass.Access.READ_ACCESS,]       | ["NetworkPolicy"]
        ["Cluster": RoleOuterClass.Access.READ_ACCESS,
         "Deployment": RoleOuterClass.Access.READ_ACCESS,
         "NetworkGraph": RoleOuterClass.Access.READ_ACCESS,
         "NetworkPolicy": RoleOuterClass.Access.READ_WRITE_ACCESS,] | ["NetworkPolicy"]
    }

    @Category(BAT)
    def "Verify token with multiple roles works as expected"() {
        when:
        "Create two roles for individual access"
        def roles = ["Indicator", "ProcessWhitelist"].collect {
            Map<String, RoleOuterClass.Access> resourceToAccess = [
                    (it): RoleOuterClass.Access.READ_ACCESS
            ]
            def role = RoleService.createRoleWithScopeAndPermissionSet("View ${it}",
                UNRESTRICTED_SCOPE_ID, resourceToAccess)
            assert RoleService.getRole(role.name)
            log.info "Created Role:\n${role.name}"
            role
        }
        assert roles.size() == 2

        and:
        "Create tokens that use either one or both roles"
        def tokens = roles.subsequences().collect {
            def token = ApiTokenService.generateToken("API Token - ${it*.name}", (it*.name).toArray(new String[0]))
            assert token != null
            token
        }
        assert tokens.size() == 3

        then:
        "Call to RPC method should succeed iff token represents union role"
        for (def token : tokens) {
            log.info "Checking behavior with token ${token.metadata.name}"
            assert canDo({
                ProcessService.getGroupedProcessByDeploymentAndContainer("unknown")
            }, token.token, true) == (token.metadata.rolesCount > 1)
        }

        and:
        "MyPermissions API should return the union of permissions"
        for (def token : tokens) {
            log.info "Checking permissions for token ${token.metadata.name}"
            assert myPermissions(token.token).resourceToAccessMap.size() == token.metadata.rolesList.size()
        }

        cleanup:
        "Revoke tokens"
        for (def token : tokens) {
            ApiTokenService.revokeToken(token.metadata.id)
        }

        and:
        "Delete roles"
        for (def role : roles) {
            RoleService.deleteRole(role.name)
        }
    }
}

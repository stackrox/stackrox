import static util.Helpers.withRetry

import java.util.concurrent.TimeUnit

import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.AuthproviderService
import io.stackrox.proto.api.v1.GroupServiceOuterClass
import io.stackrox.proto.storage.AuthProviderOuterClass.AuthProvider
import io.stackrox.proto.storage.GroupOuterClass.GroupProperties
import io.stackrox.proto.storage.GroupOuterClass.Group
import io.stackrox.proto.storage.IntegrationHealthOuterClass.IntegrationHealth.Status
import io.stackrox.proto.storage.RoleOuterClass.Access
import io.stackrox.proto.storage.RoleOuterClass.PermissionSet
import io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope
import io.stackrox.proto.storage.RoleOuterClass.Role
import io.stackrox.proto.storage.TraitsOuterClass.Traits

import util.Env

import services.AuthProviderService
import services.GroupService
import services.IntegrationHealthService
import services.RoleService

import org.junit.Rule
import org.junit.rules.Timeout
import spock.lang.IgnoreIf
import spock.lang.Retry
import spock.lang.Tag

@Retry(count = 0)
// TODO(ROX-16008): Remove this once the declarative config feature flag is enabled by default.
@IgnoreIf({ Env.get("ROX_DECLARATIVE_CONFIGURATION", "false") == "false" })
class DeclarativeConfigTest extends BaseSpecification {
    static final private String DEFAULT_NAMESPACE = "stackrox"

    static final private String CONFIGMAP_NAME = "declarative-configurations"

    // The keys are used within the config map to indicate the specific resources.
    static final private String PERMISSION_SET_KEY = "permission-set"
    static final private String ACCESS_SCOPE_KEY = "access-scope"
    static final private String ROLE_KEY = "role"
    static final private String AUTH_PROVIDER_KEY = "auth-provider"

    static final private int CREATED_RESOURCES = 6

    // Values used within testing for permission sets.
    // These include:
    //  - a valid permission set YAML (valid == upserting these will work)
    //  - a valid permission set proto object (based on the values defined in the previous YAML)
    //  - an invalid permission set YAML (invalid == failure during upserting the generated proto from these values)
    static final private String VALID_PERMISSION_SET_YAML = """\
name: ${PERMISSION_SET_KEY}
description: declarative permission set used in testing
resources:
- resource: Integration
  access: READ_ACCESS
- resource: Administration
  access: READ_ACCESS
- resource: Access
  access: READ_ACCESS
"""
    static final private VALID_PERMISSION_SET = PermissionSet.newBuilder()
            .setName(PERMISSION_SET_KEY)
            .setDescription("declarative permission set used in testing")
            .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE))
            .putAllResourceToAccess([
                    "Integration": Access.READ_ACCESS,
                    "Access": Access.READ_ACCESS,
                    "Administration": Access.READ_ACCESS,
            ]).build()
    static final private String INVALID_PERMISSION_SET_YAML = """\
name: ${PERMISSION_SET_KEY}
description: invalid declarative permission set used in testing
resources:
- resource: non-existent-resource
  access: READ_ACCESS
"""

    // Values used within testing for access scopes.
    // These include:
    //  - a valid access scope YAML (valid == upserting these will work)
    //  - a valid access scope proto object (based on the values defined in the previous YAML)
    //  - an invalid access scope YAML (invalid == failure during upserting the generated proto from these values)
    static final private String VALID_ACCESS_SCOPE_YAML = """\
name: ${ACCESS_SCOPE_KEY}
description: declarative access scope used in testing
rules:
  included:
  - cluster: remote
"""
    static final private VALID_ACCESS_SCOPE = SimpleAccessScope.newBuilder()
            .setName(ACCESS_SCOPE_KEY)
            .setDescription("declarative access scope used in testing")
            .setRules(
                    SimpleAccessScope.Rules.newBuilder()
                            .addAllIncludedClusters(["remote"])
            )
            .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE))
            .build()
    static final private String INVALID_ACCESS_SCOPE_YAML = """\
name: ${ACCESS_SCOPE_KEY}
description: invalid declarative access scope used in testing
rules:
  included:
  - cluster: remote
  clusterLabelSelectors:
  - requirements:
    - key: a
      operator: IN
"""

    // Values used within testing for roles.
    // These include:
    //  - a valid role YAML (valid == upserting these will work)
    //  - a valid role proto object (based on the values defined in the previous YAML)
    //  - an invalid role YAML (invalid == failure during upserting the generated proto from these values)
    static final private String VALID_ROLE_YAML = """\
name: ${ROLE_KEY}
description: declarative role used in testing
permissionSet: ${PERMISSION_SET_KEY}
accessScope: ${ACCESS_SCOPE_KEY}
"""
    static final private VALID_ROLE = Role.newBuilder()
            .setName(ROLE_KEY)
            .setDescription("declarative role used in testing")
            .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE))
            .build()
    static final private String INVALID_ROLE_YAML = """\
name: ${ROLE_KEY}
description: invalid declarative role used in testing
permissionSet: non-existent-permission-set
accessScope: ${ACCESS_SCOPE_KEY}
"""

    // Values used within testing for auth providers.
    // These include:
    //  - a valid auth provider YAML (valid == upserting these will work)
    //  - a valid auth provider proto object (based on the values defined in the previous YAML)
    //  - two valid group proto objects (based on the values defined in the previous YAML)
    //  - an invalid auth provider YAML (invalid == failure during upserting the generated proto from these values)
    static  final private String VALID_AUTH_PROVIDER_YAML = """\
name: ${AUTH_PROVIDER_KEY}
minimumRole: "None"
uiEndpoint: localhost:8000
groups:
- key: "email"
  value: "someone@example.com"
  role: "Admin"
oidc:
  issuer: sso.redhat.com/auth/realms/redhat-external
  mode: fragment
  clientID: SOMECLIENTID
"""
    static final private VALID_AUTH_PROVIDER = AuthProvider.newBuilder()
        .setName(AUTH_PROVIDER_KEY)
        .setUiEndpoint("localhost:8000")
        .setActive(true)
        .setEnabled(true)
        .setType("oidc")
        .putAllConfig(
                ["issuer": "https://sso.redhat.com/auth/realms/redhat-external",
                 "mode": "fragment",
                 "client_id": "SOMECLIENTID",
                 "client_secret": "",
                ])
        .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE))
        .build()

    static final private VALID_DEFAULT_GROUP = Group.newBuilder()
        .setRoleName("None")
        .setProps(GroupProperties.newBuilder()
                .setKey("")
                .setValue("")
                .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE)))
        .build()

    static final private VALID_DECLARATIVE_GROUP = Group.newBuilder()
        .setRoleName("Admin")
        .setProps(GroupProperties.newBuilder()
                .setKey("email")
                .setValue("someone@example.com")
                .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE))
        )
        .build()
    static  final private String INVALID_AUTH_PROVIDER_YAML = """\
name: ${AUTH_PROVIDER_KEY}
minimumRole: "None"
uiEndpoint: localhost:8000
groups:
- key: "email"
  value: "someone@example.com"
  role: "Admin"
oidc:
  issuer: example.com
  mode: fragment
  clientID: SOMECLIENTID
"""

    // Overwrite the default timeout, as these tests may take longer than 800 seconds to finish.
    @Rule
    @SuppressWarnings(["JUnitPublicProperty"])
    Timeout globalTimeout = new Timeout(1200, TimeUnit.SECONDS)

    def cleanup() {
        orchestrator.deleteConfigMap(CONFIGMAP_NAME, DEFAULT_NAMESPACE)
    }

    @Tag("BAT")
    def "Check successful creation, update, and deletion of declarative resources"() {
        when:

        createDefaultSetOfResources(CONFIGMAP_NAME, DEFAULT_NAMESPACE)

        then:
        // Retry this multiple times, with a pause of 60 seconds.
        // It may take some time until a) the config map contents are mapped within the pod b) the reconciliation
        // has been triggered.
        // If the tests are flaky, we have to increase this value.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            // Expect 6 integration health status for the created resources and one for the config map.
            assert response.integrationHealthCount == CREATED_RESOURCES + 1
            for (integrationHealth in response.integrationHealthList) {
                assert integrationHealth.hasLastTimestamp()
                assert integrationHealth.getErrorMessage() == ""
                assert integrationHealth.getStatus() == Status.HEALTHY
            }
        }

        // Verify the permission set is created successfully, and does specify the origin declarative.
        def permissionSet = verifyDeclarativePermissionSet(VALID_PERMISSION_SET)
        assert permissionSet

        // Verify the access scope is created successfully, and does specify the origin declarative.
        def accessScope = verifyDeclarativeAccessScope(VALID_ACCESS_SCOPE)
        assert accessScope

        // Verify the role is created successfully, and does specify the origin declarative.
        assert verifyDeclarativeRole(VALID_ROLE, permissionSet.getId(), accessScope.getId())

        // Verify the auth provider is created successfully, and does specify the origin declarative.
        def authProvider = verifyDeclarativeAuthProvider(VALID_AUTH_PROVIDER)
        assert authProvider

        // Verify the groups are created successfully, and specify the origin declarative.
        def expectedGroups = [VALID_DEFAULT_GROUP, VALID_DECLARATIVE_GROUP]
        def foundGroups = verifyDeclarativeGroups(authProvider.getId(), expectedGroups)
        assert foundGroups == expectedGroups.size() :
                "expected to find ${expectedGroups.size()} groups, but only found ${foundGroups}"

        when:
        // Update the config map to contain an invalid permission set YAML.
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, PERMISSION_SET_KEY, INVALID_PERMISSION_SET_YAML)

        then:
        // Verify the integration health for the permission set is unhealthy and contains an error message.
        // The errors will be surface after at least three consecutive occurrences, hence we need to retry multiple
        // times here.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def permissionSetHealth = response.getIntegrationHealthList().find {
                it.getName().contains(PERMISSION_SET_KEY)
            }
            assert permissionSetHealth
            assert permissionSetHealth.getErrorMessage()
            assert permissionSetHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the permission set stored is still the same.
        assert verifyDeclarativePermissionSet(VALID_PERMISSION_SET)

        when:
        // Update the config map to contain an invalid access scope YAML.
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ACCESS_SCOPE_KEY, INVALID_ACCESS_SCOPE_YAML)

        then:
        // Verify the integration health for the access scope is unhealthy and contains an error message.
        // The errors will be surface after at least three consecutive occurrences, hence we need to retry multiple
        // times here.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def accessScopeHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ACCESS_SCOPE_KEY)
            }
            assert accessScopeHealth
            assert accessScopeHealth.getErrorMessage()
            assert accessScopeHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the access scope stored is still the same.
        assert verifyDeclarativeAccessScope(VALID_ACCESS_SCOPE)

        when:
        // Update the config map to contain an invalid role YAML.
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ROLE_KEY, INVALID_ROLE_YAML)

        then:
        // Verify the integration health for the role is unhealthy and contains an error message.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def roleHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ROLE_KEY)
            }
            assert roleHealth
            assert roleHealth.getErrorMessage()
            assert roleHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the role stored is still the same.
        assert verifyDeclarativeRole(VALID_ROLE, permissionSet.getId(), accessScope.getId())

        when:
        // Update the config map to contain an invalid auth provider YAML.
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, AUTH_PROVIDER_KEY, INVALID_AUTH_PROVIDER_YAML)

        then:
        // Verify the integration health for the auth provider is unhealthy and contains an error message.
        // The errors will be surface after at least three consecutive occurrences, hence we need to retry multiple
        // times here.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def roleHealth = response.getIntegrationHealthList().find {
                it.getName().contains(AUTH_PROVIDER_KEY)
            }
            assert roleHealth
            assert roleHealth.getErrorMessage()
            assert roleHealth.getStatus() == Status.UNHEALTHY
        }

        // The previously created auth provider should not exist anymore.
        // TODO(ROX-16007): This currently is the behavior since within update we call delete + create.
        //              Maybe we should just switch to using registry.Update(), if possible.
        assert AuthProviderService.getAuthProviderService().
                getAuthProviders(
                        AuthproviderService.GetAuthProvidersRequest.newBuilder()
                                .setName(VALID_AUTH_PROVIDER.getName()).build()
                )
                .getAuthProvidersCount() == 0

        when:
        orchestrator.deleteConfigMap(CONFIGMAP_NAME, DEFAULT_NAMESPACE)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            assert response.getIntegrationHealthCount() == 1
            def configMapHealth = response.getIntegrationHealth(0)
            assert configMapHealth
            assert configMapHealth.getName().contains("Config Map")
            assert configMapHealth.getErrorMessage() == ""
            assert configMapHealth.getStatus() == Status.HEALTHY
        }

        // The previously created permission set should not exist anymore.
        def permissionSetAfterDeletion = RoleService.getRoleService().listPermissionSets()
                .getPermissionSetsList().find {
                    it.getName() == VALID_PERMISSION_SET.getName()
                }
        assert permissionSetAfterDeletion == null

        // The previously created access scope should not exist anymore.
        def accessScopeAfterDeletion = RoleService.getRoleService()
                .listSimpleAccessScopes()
                .getAccessScopesList().find {
                    it.getName() == VALID_ACCESS_SCOPE.getName()
                }
        assert accessScopeAfterDeletion == null

        // The previously created role should not exist anymore.
        try {
            RoleService.getRole(VALID_ROLE.getName())
        } catch (StatusRuntimeException ex) {
            assert ex.getStatus().getCode() == io.grpc.Status.NOT_FOUND.getCode()
        }

        // The previously created auth provider should not exist anymore.
        assert AuthProviderService.getAuthProviderService().
                getAuthProviders(
                        AuthproviderService.GetAuthProvidersRequest.newBuilder()
                                .setName(VALID_AUTH_PROVIDER.getName()).build()
                )
                .getAuthProvidersCount() == 0

        // The previously created groups should not exist anymore.
        assert GroupService.getGroups(
                GroupServiceOuterClass.GetGroupsRequest.newBuilder().setAuthProviderId(authProvider.getId()).build())
                .getGroupsCount() == 0
    }

    @Tag("BAT")
    def "Check creating invalid configuration will not work"() {
        when:
        orchestrator.createConfigMap(CONFIGMAP_NAME,
                [
                        (PERMISSION_SET_KEY): INVALID_PERMISSION_SET_YAML,
                        (ACCESS_SCOPE_KEY): INVALID_ACCESS_SCOPE_YAML,
                        (ROLE_KEY): INVALID_ROLE_YAML,
                        (AUTH_PROVIDER_KEY): INVALID_AUTH_PROVIDER_YAML,
                ], DEFAULT_NAMESPACE)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            // Expect 6 integration health status for the created resources and one for the config map.
            assert response.integrationHealthCount == CREATED_RESOURCES + 1

            for (integrationHealth in response.getIntegrationHealthList()) {
                // Config map health will be healthy and do not indicate an error.
                if (integrationHealth.getName().contains("Config Map")) {
                    assert integrationHealth
                    assert integrationHealth.hasLastTimestamp()
                    assert integrationHealth.getErrorMessage() == ""
                    assert integrationHealth.getStatus() == Status.HEALTHY
                } else {
                    assert integrationHealth.hasLastTimestamp()
                    assert integrationHealth.getErrorMessage()
                    assert integrationHealth.getStatus() == Status.UNHEALTHY
                }
            }
        }

        // No permission set should be created.
        def permissionSetAfterDeletion = RoleService.getRoleService().listPermissionSets()
                .getPermissionSetsList().find {
            it.getName() == VALID_PERMISSION_SET.getName()
        }
        assert permissionSetAfterDeletion == null

        // No access scope should be created.
        def accessScopeAfterDeletion = RoleService.getRoleService()
                .listSimpleAccessScopes()
                .getAccessScopesList().find {
            it.getName() == VALID_ACCESS_SCOPE.getName()
        }
        assert accessScopeAfterDeletion == null

        // No role should be created.
        try {
            RoleService.getRole(VALID_ROLE.getName())
        } catch (StatusRuntimeException ex) {
            assert ex.getStatus().getCode() == io.grpc.Status.NOT_FOUND.getCode()
        }

        // No auth provider should be created.
        assert AuthProviderService.getAuthProviderService().
                getAuthProviders(
                        AuthproviderService.GetAuthProvidersRequest.newBuilder()
                                .setName(VALID_AUTH_PROVIDER.getName()).build()
                )
                .getAuthProvidersCount() == 0

        when:
        orchestrator.deleteConfigMap(CONFIGMAP_NAME, DEFAULT_NAMESPACE)

        then:
        // Only the config map health status should exist, all others should be removed.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            assert response.getIntegrationHealthCount() == 1
            def configMapHealth = response.getIntegrationHealth(0)
            assert configMapHealth
            assert configMapHealth.getName().contains("Config Map")
            assert configMapHealth.getErrorMessage() == ""
            assert configMapHealth.getStatus() == Status.HEALTHY
        }
    }

    @Tag("BAT")
    def "Check orphaned declarative resources are correctly handled"() {
        when:

        createDefaultSetOfResources(CONFIGMAP_NAME, DEFAULT_NAMESPACE)

        then:
        // Retry this multiple times, with a pause of 60 seconds.
        // It may take some time until a) the config map contents are mapped within the pod b) the reconciliation
        // has been triggered.
        // If the tests are flaky, we have to increase this value.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            // Expect 6 integration health status for the created resources and one for the config map.
            assert response.integrationHealthCount == CREATED_RESOURCES + 1
            for (integrationHealth in response.integrationHealthList) {
                assert integrationHealth.hasLastTimestamp()
                assert integrationHealth.getErrorMessage() == ""
                assert integrationHealth.getStatus() == Status.HEALTHY
            }
        }

        when:
        deleteConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, PERMISSION_SET_KEY)

        then:
        // Verify the integration health for the permission set is unhealthy and contains an error message.
        // The errors will be surface after at least three consecutive occurrences, hence we need to retry multiple
        // times here.
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def permissionSetHealth = response.getIntegrationHealthList().find {
                it.getName().contains(PERMISSION_SET_KEY)
            }
            assert permissionSetHealth
            assert permissionSetHealth.getErrorMessage().contains("referenced by another object")
            assert permissionSetHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the permission set stored is still the same, but origin is orphaned.
        assert verifyDeclarativePermissionSet(VALID_PERMISSION_SET.toBuilder()
                    .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE_ORPHANED))
                    .build()
        )

        when:
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, PERMISSION_SET_KEY, VALID_PERMISSION_SET_YAML)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def permissionSetHealth = response.getIntegrationHealthList().find {
                it.getName().contains(PERMISSION_SET_KEY)
            }
            assert permissionSetHealth
            assert permissionSetHealth.hasLastTimestamp()
            assert permissionSetHealth.getErrorMessage() == ""
            assert permissionSetHealth.getStatus() == Status.HEALTHY
        }

        when:
        deleteConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ACCESS_SCOPE_KEY)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def accessScopeHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ACCESS_SCOPE_KEY)
            }
            assert accessScopeHealth
            assert accessScopeHealth.getErrorMessage().contains("referenced by another object")
            assert accessScopeHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the access scope stored is still the same, but origin is orphaned.
        assert verifyDeclarativeAccessScope(VALID_ACCESS_SCOPE.toBuilder()
                .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE_ORPHANED))
                .build()
        )

        when:
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ACCESS_SCOPE_KEY, VALID_ACCESS_SCOPE_YAML)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def accessScopeHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ACCESS_SCOPE_KEY)
            }
            assert accessScopeHealth
            assert accessScopeHealth.hasLastTimestamp()
            assert accessScopeHealth.getErrorMessage() == ""
            assert accessScopeHealth.getStatus() == Status.HEALTHY
        }

        when:
        def authProvidersResponse = AuthProviderService.getAuthProviders()
        def authProvider = authProvidersResponse.getAuthProvidersList().find {
            it.getName() == AUTH_PROVIDER_KEY
        }
        def imperativeGroup = Group.newBuilder()
                .setRoleName(ROLE_KEY)
                .setProps(GroupProperties.newBuilder()
                    .setAuthProviderId(authProvider.getId())
                    .setKey("white")
                    .setValue("stripes"))
                .build()
        GroupService.createGroup(imperativeGroup)
        def imperativeGroupWithId = GroupService.getGroups(GroupServiceOuterClass.GetGroupsRequest.newBuilder()
                .setAuthProviderId(authProvider.getId())
                .setKey("white")
                .setValue("stripes")
                .build())
                .getGroups(0)

        deleteConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ROLE_KEY)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def roleHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ROLE_KEY)
            }
            assert roleHealth
            assert roleHealth.getErrorMessage().contains("is referenced by groups")
            assert roleHealth.getStatus() == Status.UNHEALTHY
        }

        // Verify the role stored is still the same, but origin is orphaned.
        assert verifyDeclarativeRole(VALID_ROLE.toBuilder()
                .setTraits(Traits.newBuilder().setOrigin(Traits.Origin.DECLARATIVE_ORPHANED))
                .build()
        )

        when:
        updateConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, ROLE_KEY, VALID_ROLE_YAML)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            def roleHealth = response.getIntegrationHealthList().find {
                it.getName().contains(ROLE_KEY)
            }
            assert roleHealth
            assert roleHealth.hasLastTimestamp()
            assert roleHealth.getErrorMessage() == ""
            assert roleHealth.getStatus() == Status.HEALTHY
        }

        when:
        deleteConfigMapValue(CONFIGMAP_NAME, DEFAULT_NAMESPACE, AUTH_PROVIDER_KEY)

        then:
        withRetry(5, 60) {
            def response = IntegrationHealthService.getDeclarativeConfigHealthInfo()
            // After auth provider deletion we should be left only with integration health for:
            // - access scope
            // - role
            // - permission set
            // - config map
            assert response.getIntegrationHealthCount() == 4
        }

        when:
        GroupService.getGroup(imperativeGroupWithId.getProps())

        then:
        // Verify imperative group referencing declarative auth provider is deleted with it.
        def error = thrown(StatusRuntimeException)
        assert error.getStatus().getCode() == io.grpc.Status.Code.NOT_FOUND
    }

    // Helpers

    // createDefaultSetOfResources creates the following resources:
    //  - permission set with valid configuration.
    //  - access scope with valid configuration.
    //  - role with valid configuration, referencing the previously created permission set / access scope.
    //  - auth provider with valid configuration, and two groups (one default, one separate)
    private createDefaultSetOfResources(String configMapName, String namespace) {
        orchestrator.createConfigMap(configMapName,
                [
                        (PERMISSION_SET_KEY): VALID_PERMISSION_SET_YAML,
                        (ACCESS_SCOPE_KEY): VALID_ACCESS_SCOPE_YAML,
                        (ROLE_KEY): VALID_ROLE_YAML,
                        (AUTH_PROVIDER_KEY): VALID_AUTH_PROVIDER_YAML,
                ], namespace)
    }

    // updateConfigMapValue updates a key / value pair within the given config map.
    private updateConfigMapValue(String configMapName, String namespace, String key, String value) {
        def configMap = orchestrator.getConfigMap(configMapName, namespace)
        configMap.data.put(key, value)
        orchestrator.createConfigMap(configMap)
    }

    private deleteConfigMapValue(String configMapName, String namespace, String key) {
        def configMap = orchestrator.getConfigMap(configMapName, namespace)
        configMap.data.remove(key)
        orchestrator.createConfigMap(configMap)
    }

    // verifyDeclarativeRole will verify that the expected role exists within the API and shares the same values.
    // The retrieved role from the API will be returned.
    private Role verifyDeclarativeRole(Role expectedRole, String permissionSetID, String accessScopeID) {
        def role = RoleService.getRole(expectedRole.getName())
        assert role : "declarative role ${expectedRole.getName()} does not exist"
        assert role.getName() == expectedRole.getName()
        assert role.getDescription() == expectedRole.getDescription()
        assert role.getTraits().getOrigin() == expectedRole.getTraits().getOrigin()
        assert role.getAccessScopeId() == accessScopeID
        assert role.getPermissionSetId() == permissionSetID
        return role
    }

    private Role verifyDeclarativeRole(Role expectedRole) {
        def role = RoleService.getRole(expectedRole.getName())
        assert role : "declarative role ${expectedRole.getName()} does not exist"
        assert role.getName() == expectedRole.getName()
        assert role.getDescription() == expectedRole.getDescription()
        assert role.getTraits().getOrigin() == expectedRole.getTraits().getOrigin()
        return role
    }

    // verifyDeclarativePermissionSet will verify that the expected permission set exists within the API and
    // shares the same values.
    // The retrieved permission set from the API will be returned, which will have the ID field populated.
    private PermissionSet verifyDeclarativePermissionSet(PermissionSet expectedPermissionSet) {
        def permissionSetsResponse = RoleService.getRoleService().listPermissionSets()
        def permissionSet = permissionSetsResponse.getPermissionSetsList().find {
            it.getName() == expectedPermissionSet.getName()
        }
        assert permissionSet
        assert permissionSet.getDescription() == expectedPermissionSet.getDescription()
        assert permissionSet.getTraits().getOrigin() == expectedPermissionSet.getTraits().getOrigin()
        assert permissionSet.getResourceToAccessMap() == expectedPermissionSet.getResourceToAccessMap()
        assert permissionSet.getId()
        return permissionSet
    }

    // verifyDeclarativeAccessScope will verify that the expected access scope exists within the API and
    // shares the same values.
    // The retrieved access scope from the API will be returned, which will have the ID field populated.
    private SimpleAccessScope verifyDeclarativeAccessScope(SimpleAccessScope expectedAccessScope) {
        def accessScopesResponse = RoleService.getRoleService().listSimpleAccessScopes()
        def accessScope = accessScopesResponse.getAccessScopesList().find {
            it.getName() == expectedAccessScope.getName()
        }
        assert accessScope
        assert accessScope.getDescription() == expectedAccessScope.getDescription()
        assert accessScope.getTraits().getOrigin() == expectedAccessScope.getTraits().getOrigin()
        assert accessScope.getRules() == expectedAccessScope.getRules()
        assert accessScope.getId()
        return accessScope
    }

    // verifyDeclarativeGroups will verify that the expected groups exist within the API and share the same
    // values.
    // The number of groups found within the list of expected groups will be returned.
    private int verifyDeclarativeGroups(String authProviderID, List<Group> expectedGroups) {
        def groupsResponse = GroupService.getGroups(
                GroupServiceOuterClass.GetGroupsRequest.newBuilder().setAuthProviderId(authProviderID).build())
        assert groupsResponse.getGroupsCount() == expectedGroups.size()

        def foundGroups = 0
        for (group in groupsResponse.getGroupsList()) {
            for (expectedGroup in expectedGroups) {
                if (group.getRoleName() == expectedGroup.getRoleName()) {
                    foundGroups++
                    assert group.getProps().getKey() == expectedGroup.getProps().getKey()
                    assert group.getProps().getValue() == expectedGroup.getProps().getValue()
                    assert group.getProps().getAuthProviderId() == authProviderID
                    assert group.getProps().getTraits().getOrigin() == expectedGroup.getProps().getTraits().getOrigin()
                }
            }
        }
        return foundGroups
    }

    // verifyDeclarativeAuthProvider will verify that the expected auth provider exists within the API and
    // shares the same values.
    // The retrieved auth provider from the API will be returned, which will have the ID field populated.
    private AuthProvider verifyDeclarativeAuthProvider(AuthProvider expectedAuthProvider) {
        def authProviderResponse = AuthProviderService.getAuthProviderService().
                getAuthProviders(
                        AuthproviderService.GetAuthProvidersRequest.newBuilder()
                                .setName(expectedAuthProvider.getName()).build()
                )
        assert authProviderResponse.getAuthProvidersCount() == 1 :
                "expected one auth provider with name ${expectedAuthProvider.getName()} but " +
                        "got ${authProviderResponse.getAuthProvidersCount()}"
        def authProvider = authProviderResponse.getAuthProviders(0)
        assert authProvider
        assert authProvider.getName() == expectedAuthProvider.getName()
        assert authProvider.getType() == expectedAuthProvider.getType()
        assert authProvider.getLoginUrl()
        assert authProvider.getUiEndpoint() == expectedAuthProvider.getUiEndpoint()
        assert authProvider.getTraits().getOrigin() == expectedAuthProvider.getTraits().getOrigin()
        assert authProvider.getActive()
        assert authProvider.getEnabled()
        assert authProvider.getConfigMap() == expectedAuthProvider.getConfigMap()
        return authProvider
    }
}

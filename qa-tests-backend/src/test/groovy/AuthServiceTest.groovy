import io.stackrox.proto.api.v1.AuthServiceOuterClass
import io.stackrox.proto.storage.RoleOuterClass

import services.AuthService
import services.BaseService

import spock.lang.Tag

@Tag("BAT")
@Tag("COMPATIBILITY")

class AuthServiceTest extends BaseSpecification {

    private static Map<String, List<String>> getAttrMap(List<AuthServiceOuterClass.UserAttribute> attrList) {
        attrList.collectEntries {
            [it.key, it.valuesList]
        }
    }

    def "Verify response for basic auth"() {
        when:
        BaseService.useBasicAuth()
        AuthServiceOuterClass.AuthStatus status = AuthService.getAuthStatus()

        then:
        assert status
        assert status.userId == "admin"

        status.authProvider.withDo {
            assert name == "Login with username/password"
            assert id == "4df1b98c-24ed-4073-a9ad-356aec6bb62d"
            assert type == "basic"
        }

        status.userInfo.withDo {
            assert permissions.resourceToAccessCount > 0
            permissions.resourceToAccessMap.each {
                assert it.value == RoleOuterClass.Access.READ_WRITE_ACCESS
            }
            def adminRole = rolesList.find { it.name == "Admin" }
            assert adminRole
        }

        def attrMap = getAttrMap(status.userAttributesList)
        assert attrMap["username"] == ["admin"]
        assert attrMap["role"] == ["Admin"]
    }

    def "Verify response for auth token"() {
        when:
        useTokenServiceAuth()

        AuthServiceOuterClass.AuthStatus status = AuthService.getAuthStatus()

        then:
        assert status
        assert status.userId.startsWith("auth-token:")
        assert !status.authProvider.id

        status.userInfo.withDo {
            assert permissions.resourceToAccessCount > 0
            permissions.resourceToAccessMap.each {
                assert it.value == RoleOuterClass.Access.READ_WRITE_ACCESS
            }

            def tokenRole = rolesList.find { it.name.startsWith("Test Automation Role - ") }
            assert tokenRole
        }

        def attrMap = getAttrMap(status.userAttributesList)
        assert attrMap["name"][0].startsWith("allAccessToken-")
        assert attrMap["role"][0].startsWith("Test Automation Role - ")
    }
}

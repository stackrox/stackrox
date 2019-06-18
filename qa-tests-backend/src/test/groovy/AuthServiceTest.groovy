
import io.stackrox.proto.api.v1.AuthServiceOuterClass
import services.AuthService
import org.apache.commons.lang.StringUtils

class AuthServiceTest extends BaseSpecification {
    def "Verify Auth Token is used"() {
    when:
        AuthServiceOuterClass.AuthStatus status = AuthService.getAuthStatus()
    then:
        assert !(StringUtils.isEmpty(status.getUserId()))
    }
}

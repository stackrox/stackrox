import static Services.getSecret
import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category

class SecretsTest extends BaseSpecification {
    @Category(BAT)
    def  "Verify the secret api can return the secrets : C964"() {
         when:
        "Create a Secret"
        String secID = orchestrator.createSecret("qasec")

         and:
        "Create a Deployment using above created secret"
        Deployment deployment = new Deployment()
                 .setName ("depwithsecrets")
                 .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                 .addLabel ( "app", "test" )
                 .addVolName("test")
                 .addVolMountName("test")
                 .addMountPath("/etc/try")
                 .addSecretName("qasec")
        orchestrator.createDeployment(deployment)

         then:
        "Verify Secret is added to the list"
        assert getSecret(secID) != null

         cleanup:
        "Remove Deployment #deploymentName"
        orchestrator.deleteDeployment(deployment)
        orchestrator.deleteSecret("qasec")
    }
}

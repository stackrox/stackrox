import static util.Helpers.withRetry

import io.stackrox.proto.storage.SecretOuterClass.Secret

import objects.Deployment
import services.SecretService
import util.Timer

import spock.lang.Tag
import spock.lang.Unroll

class SecretsTest extends BaseSpecification {

    private static Deployment renderDeployment(String deploymentName, String secretName, boolean fromEnv) {
        Deployment deploy = new Deployment()
                .setName (deploymentName)
                .setNamespace("qa")
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                .addLabel ( "app", "test" )
                .setCommand(["sleep", "600"])
        if (fromEnv) {
            deploy.setEnvFromSecrets([secretName])
        } else {
            deploy.addVolume("test", "/etc/try")
                    .addSecretName("test", secretName)
        }
        return deploy
    }

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    
    def "Verify the secret api can return the secret's information when adding a new secret"() {
        when:
        "Create a Secret"
        String secretName = "qasec"
        String secID = orchestrator.createSecret(secretName)

        then:
        "Verify Secret is added to the list"
        assert SecretService.getSecret(secID) != null

        cleanup:
        "Remove Secret #secretName"
        orchestrator.deleteSecret(secretName)
    }

    @Unroll
    @Tag("BAT")
    
    def "Verify the secret item should show the binding deployments (from env var: #fromEnv)"() {
        when:
        "Create a Secret"
        String secretName = "qasec"
        String secID = orchestrator.createSecret("qasec")

        and:
        "Create a Deployment using above created secret"
        String deploymentName = "depwithsecrets"
        Deployment deployment = renderDeployment(deploymentName, secretName, fromEnv)
        orchestrator.createDeployment(deployment)

        then:
        "Verify the deployment is binding with the secret"
        assert SecretService.getSecret(secID) != null
        Set<String> secretSet = orchestrator.getDeploymentSecrets(deployment)
        assert secretSet.contains(secretName)

        cleanup:
        "Remove Secret #secretName and Deployment #deploymentName"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
        orchestrator.deleteSecret(secretName)

        where:
        "Data inputs are"
        fromEnv << [false, true]
    }

    @Unroll
    @Tag("BAT")
    
    def "Verify the secret should not show the deleted binding deployment (from env var: #fromEnv)"() {
        when:
        "Create a Secret and bind deployment with it"
        String secretName = "qasec"
        String deploymentName = "depwithsecrets"
        String secID = orchestrator.createSecret("qasec")
        Deployment deployment = renderDeployment(deploymentName, secretName, fromEnv)

        orchestrator.createDeployment(deployment)

        def timer = new Timer(30, 1)
        def found = false
        while (!found && timer.IsValid()) {
            Secret secretInfo = SecretService.getSecret(secID)

            def match = secretInfo.relationship.deploymentRelationshipsList.find { it.id == deployment.deploymentUid }
            found = match != null
        }
        assert found : "Secret-to-deployment relationship not found"

        and:
        "Delete the binding deployment"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        and:
        "Wait until the binding deployment is gone from the secret"
        timer = new Timer(10, 3)

        //Add waiting logic cause stackrox need some time to response the number of deployments' change
        found = true
        while (found && timer.IsValid()) {
            def secretUpdate = SecretService.getSecret(secID)

            def match = secretUpdate.relationship.deploymentRelationshipsList.find { it.id == deployment.deploymentUid }
            found = match != null
        }

        then:
        "The Secret-to-deployment relationship should no longer exist"
        assert !found : "Secret-to-deployment relationship still exists"

        cleanup:
        "Remove Secret #secretName"
        orchestrator.deleteSecret(secretName)

        where:
        "Data inputs are"
        fromEnv << [false, true]
    }

    @Unroll
    @Tag("BAT")
    
    def "Verify the secret information should not be infected by the previous secrets (from env var: #fromEnv)"() {
        when:
        "Create a Secret and bind deployment with it"
        String secretName = "qasec"
        String deploymentName = "depwithsecrets"

        String secID = orchestrator.createSecret(secretName)
        Deployment deployment = renderDeployment(deploymentName, secretName, fromEnv)
        orchestrator.createDeployment(deployment)

        and:
        "Delete this deployment and create another deployment binding with the secret name with different name"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        String deploymentSecName = "depwithsecretssec"
        Deployment deploymentSec = renderDeployment(deploymentSecName, secretName, fromEnv)
        orchestrator.createDeployment(deploymentSec)

        then:
        "Verify the secret should show the new bounding deployment"
        withRetry(30, 1) {
            Secret secretInfo = SecretService.getSecret(secID)
            assert secretInfo.getRelationship().getDeploymentRelationshipsCount() == 1
            assert secretInfo.getRelationship().getDeploymentRelationships(0).getName() == deploymentSecName
        }

        cleanup:
        "Remove Deployment #deploymentName and Secret #secretName"
        orchestrator.deleteAndWaitForDeploymentDeletion(deploymentSec)
        orchestrator.deleteSecret(secretName)

        where:
        "Data inputs are"
        fromEnv << [false, true]
    }

    @Unroll
    @Tag("BAT")
    
    def "Verify secrets page should not be messed up when a deployment's secret changed (from env var: #fromEnv)"() {
        when:
        "Create a Secret and bind deployment with it"
        String secretNameOne = "qasec1"
        String deploymentNameOne = "depwithsecrets1"
        String secIDOne = orchestrator.createSecret("qasec1")
        Deployment deploymentOne = renderDeployment(deploymentNameOne, secretNameOne, fromEnv)
        orchestrator.createDeployment(deploymentOne)

        String secretNameTwo = "qasec2"
        String deploymentNameTwo = "depwithsecrets2"
        String secIDTwo = orchestrator.createSecret("qasec2")
        Deployment deploymentTwo = renderDeployment(deploymentNameTwo, secretNameTwo, fromEnv)
        orchestrator.createDeployment(deploymentTwo)

        and:
        "Delete this deployment and create another deployment binding with the secret name with different name"
        orchestrator.deleteAndWaitForDeploymentDeletion(deploymentOne, deploymentTwo)

        deploymentOne = renderDeployment(deploymentNameOne, secretNameTwo, fromEnv)
        deploymentTwo = renderDeployment(deploymentNameTwo, secretNameOne, fromEnv)
        orchestrator.createDeployment(deploymentOne)
        orchestrator.createDeployment(deploymentTwo)

        then:
        "Verify the secret should show the new bounding deployment"
        Secret secretInfoOne = SecretService.getSecret(secIDOne)
        Secret secretInfoTwo = SecretService.getSecret(secIDTwo)

        assert secretInfoOne.getRelationship().getDeploymentRelationships(0).getName() == deploymentNameTwo
        assert secretInfoTwo.getRelationship().getDeploymentRelationships(0).getName() == deploymentNameOne

        cleanup:
        "Remove Deployment and Secret"
        orchestrator.deleteAndWaitForDeploymentDeletion(deploymentOne, deploymentTwo)
        orchestrator.deleteSecret(secretNameOne)
        orchestrator.deleteSecret(secretNameTwo)

        where:
        "Data inputs are"
        fromEnv << [false, true]
    }
}

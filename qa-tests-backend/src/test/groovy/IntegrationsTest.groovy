import groups.Notifiers
import io.fabric8.kubernetes.client.LocalPortForward
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.NotifierOuterClass
import groups.BAT
import groups.Integration
import io.stackrox.proto.storage.ScopeOuterClass
import objects.EmailNotifier
import objects.GenericNotifier
import objects.JiraNotifier
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Notifier
import objects.PagerDutyNotifier
import objects.SlackNotifier
import objects.SplunkNotifier
import objects.TeamsNotifier
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.CreatePolicyService
import services.ExternalBackupService
import services.ImageIntegrationService
import services.NetworkPolicyService
import spock.lang.Unroll
import objects.Deployment
import objects.Service
import util.Env
import common.Constants

class IntegrationsTest extends BaseSpecification {
    static final private String NOTIFIERDEPLOYMENT = "netpol-notification-test-deployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(NOTIFIERDEPLOYMENT)
                    .setImage("nginx")
                    .addLabel("app", NOTIFIERDEPLOYMENT),
    ]

    private static final CA_CERT = '''-----BEGIN CERTIFICATE-----
MIIDgDCCAmgCCQDYOU2KIlcBQjANBgkqhkiG9w0BAQsFADCBgTELMAkGA1UEBhMC
VVMxCzAJBgNVBAgMAkNBMQswCQYDVQQHDAJTRjERMA8GA1UECgwIc3RhY2tyb3gx
HzAdBgNVBAMMFndlYmhvb2tzZXJ2ZXIuc3RhY2tyb3gxJDAiBgkqhkiG9w0BCQEW
FXN0YWNrcm94QHN0YWNrcm94LmNvbTAeFw0xOTAzMjMxNTQzMjVaFw0yOTAzMjAx
NTQzMjVaMIGBMQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExCzAJBgNVBAcMAlNG
MREwDwYDVQQKDAhzdGFja3JveDEfMB0GA1UEAwwWd2ViaG9va3NlcnZlci5zdGFj
a3JveDEkMCIGCSqGSIb3DQEJARYVc3RhY2tyb3hAc3RhY2tyb3guY29tMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuPzgVGykTALNHDljDiCjwI4ZfF2r
lGKWdtvUhurh42Cl2Kfn0Vgy7mYRjdK/uOiSIl6LVXuNw7w4yg48dXm8By+I3+hs
vMH4ixykWxPn6Ez3Utuuwggn/yAs4kE2Wj0ztFMpRHBGL7Qi7oEv+Vo4349ZJg16
a55db45O3LgOED119F1hQvxblNZhcA2hnNOhveXsJLfdOQKz6UA4KtdBFXxEeZuB
fC45wCHw6kjRrBEPYKB4py4ywYMdUHqswBDn6B3LtwvrrJVPTySK4sgZmOTF2XGg
JRm52MS0rYEvBpEtgkPdknoIv0VnxihMUuRhMXHfGOTFyhWuf/nF2aihXwIDAQAB
MA0GCSqGSIb3DQEBCwUAA4IBAQCYT7jo6Durhx+liRgYNO3G3mRyc36syVNGsllU
Mf5wOUHjxWfppWHzldxMeZRKksrg7xfMXdcGaOOZgD8Ir/pPK2HP48g6KIDWCiVO
kh9AGCLY9osxkBqAihtvJWNkEda+wA9ggF/7wx+0Ci+b/1NvXHeNU3uO3rP7Npwc
rxhvyNqv7MwqpMN6V8hFxqM/3ny8aoUedFsYsEvm8Dm1VLyBiIqZk0CA2oj3NIjb
ObOdSTZUQI4TZOXOpJCpa97CnqroNi7RrT05JOfoe/DPmhoJmF4AUrnd/YUb8pgF
/jvC1xBvPVtJFbYeBVysQCrRk+f/NyyUejQv+OCJ+B1KtJh4
-----END CERTIFICATE-----'''

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        DEPLOYMENTS.each { Services.waitForDeployment(it) }
    }

    def cleanupSpec() {
        DEPLOYMENTS.each { orchestrator.deleteDeployment(it) }
    }

    @Unroll
    @Category([BAT])
    def "Verify create Email Integration (port #port, disable TLS=#disableTLS, startTLS=#startTLS)"() {
        given:
        "a configuration that is expected to work"
        EmailNotifier notifier = new EmailNotifier("Email Test", disableTLS, startTLS, port)

        when:
        "the integration is tested"
        Boolean response = notifier.testNotifier()

        then:
        "the API should return an empty message or an error, depending on the config"
        assert response == shouldSucceed

        where:
        "data"

        port | disableTLS | startTLS | shouldSucceed

        // Port 465 tests
        // This port speaks TLS from the start.
        // (Also test null, since 465 is the default.)
        /////////////////
        // Speaking TLS should work
        465  | false      | NotifierOuterClass.Email.AuthMethod.DISABLED   | true
        null | false      | NotifierOuterClass.Email.AuthMethod.DISABLED   | true

        // Speaking non-TLS to a TLS port should fail and not time out, regardless of STARTTLS (see ROX-366)
        465  | true       | NotifierOuterClass.Email.AuthMethod.DISABLED   | false
        465  | true       | NotifierOuterClass.Email.AuthMethod.PLAIN      | false
        null | true       | NotifierOuterClass.Email.AuthMethod.DISABLED   | false
        null | true       | NotifierOuterClass.Email.AuthMethod.PLAIN      | false

        // Port 587 tests
        // At MailGun, this port begins unencrypted and supports STARTTLS.
        /////////////////
        // Starting unencrypted and _not_ using STARTTLS should work
        587  | true       | NotifierOuterClass.Email.AuthMethod.DISABLED | true
        // Starting unencrypted and using STARTTLS should work
        587  | true       | NotifierOuterClass.Email.AuthMethod.PLAIN    | true
        587  | true       | NotifierOuterClass.Email.AuthMethod.LOGIN    | true
        // Speaking TLS to a non-TLS port should fail whether you use STARTTLS or not.
        587  | false      | NotifierOuterClass.Email.AuthMethod.DISABLED | false

        // Cannot add port 25 tests since GCP blocks outgoing
        // connections to port 25
    }

    @Unroll
    @Category(BAT)
    def "Verify create Generic Integration Test Endpoint (#tlsOptsDesc, audit=#auditLoggingEnabled)"() {
        when:
        "the integration is tested"
        GenericNotifier notifier = new GenericNotifier(
                "Generic Test",
                enableTLS,
                caCert,
                skipTLSVerification,
                auditLoggingEnabled
        )

        then :
        "the API should return an empty message or an error, depending on the config"
        assert shouldSucceed == notifier.testNotifier()

        where:
        "data"

        enableTLS | caCert | skipTLSVerification | auditLoggingEnabled | shouldSucceed | tlsOptsDesc

        false | ""         | false               | false | true | "no TLS"
        true  | ""         | true                | false | true | "TLS, no verify"
        true  | CA_CERT    | false               | false | true | "TLS, verify custom CA"
        true  | ""         | false               | false | false | "TLS, verify system CA"
        false | ""         | false               | true | true | "no TLS"
        true  | ""         | true                | true | true | "TLS, no verify"
        true  | CA_CERT    | false               | true | true | "TLS, verify custom CA"
        true  | ""         | false               | true | false | "TLS, verify system CA"
    }

    @Unroll
    @Category(Integration)
    def "Verify Splunk Integration (legacy mode: #legacy)"() {
        given:
        "Assume cluster is not OS"
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        and:
        "the integration is tested"
        def uid = UUID.randomUUID()
        def deploymentName = "splunk-${uid}"
        orchestrator.createImagePullSecret("qa-stackrox", Env.mustGetDockerIOUserName(),
                 Env.mustGetDockerIOPassword(), Constants.ORCHESTRATOR_NAMESPACE)
        Deployment deployment =
            new Deployment()
                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                .setName(deploymentName)
                .setImage("stackrox/splunk-test-repo:6.6.0")
                .addPort (8000)
                .addPort (8088)
                .addPort(8089)
                .addAnnotation("test", "annotation")
                .setEnv([ "SPLUNK_START_ARGS": "--accept-license", "SPLUNK_USER": "root" ])
                .addLabel("app", deploymentName)
                .setPrivilegedFlag(true)
                .addVolume("test", "/tmp")
                .addImagePullSecret("qa-stackrox")
        orchestrator.createDeployment(deployment)

        Service httpSvc = new Service("splunk-http-${uid}", Constants.ORCHESTRATOR_NAMESPACE)
                 .addLabel("app", deploymentName)
                 .addPort(8000, "TCP")
                 .setType(Service.Type.CLUSTERIP)
        orchestrator.createService(httpSvc)

        Service  collectorSvc = new Service("splunk-collector-${uid}", Constants.ORCHESTRATOR_NAMESPACE)
                .addLabel("app", deploymentName)
                .addPort(8088, "TCP")
                .setType(Service.Type.CLUSTERIP)
        orchestrator.createService(collectorSvc)

        Service httpsSvc = new Service("splunk-https-${uid}", Constants.ORCHESTRATOR_NAMESPACE)
                .addLabel("app", deploymentName)
                .addPort(8089, "TCP")
                .setType(Service.Type.CLUSTERIP)
        orchestrator.createService(httpsSvc)

        LocalPortForward splunkPortForward = orchestrator.createPortForward(8089, deployment)

        when:
        "call the grpc API for the splunk integration."
        SplunkNotifier notifier = new SplunkNotifier(legacy, collectorSvc.name, splunkPortForward.localPort)
        notifier.createNotifier()

        and:
        "Edit the policy with the latest keyword."
        PolicyOuterClass.Policy.Builder policy = Services.getPolicyByName("Latest tag").toBuilder()

        def nginxName = "nginx-spl-violation"
        policy.setName("${policy.name} ${uid}")
              .setId("") // set ID to empty so that a new policy is created and not overwrite the original latest tag
              .addScope(ScopeOuterClass.Scope.newBuilder()
                .setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                  .setKey("app")
                  .setValue(nginxName)))
              .addNotifiers(notifier.getId())
        def policyId = CreatePolicyService.createNewPolicy(policy.build())

        and:
        "Create a new deployment to trigger the violation against the policy"
        Deployment nginxdeployment =
                new Deployment()
                        .setName(nginxName)
                        .setImage("nginx:latest")
                        .addLabel("app", nginxName)
        orchestrator.createDeployment(nginxdeployment)
        assert Services.waitForViolation(nginxName, policy.name, 60)

        then:
        "Verify the messages are seen in the json"
        notifier.validateViolationNotification(policy.build(), nginxdeployment, strictIntegrationTesting)

        cleanup:
        "remove Deployment and services"
        if (deployment != null) {
            orchestrator.deleteDeployment(deployment)
            orchestrator.deleteDeployment(nginxdeployment)
        }
        orchestrator.deleteService(httpSvc.name, httpSvc.namespace)
        orchestrator.deleteService(collectorSvc.name, collectorSvc.namespace)
        orchestrator.deleteService(httpsSvc.name, httpsSvc.namespace)
        orchestrator.deleteSecret("qa-stackrox", Constants.ORCHESTRATOR_NAMESPACE)
        if (policy != null) {
            CreatePolicyService.deletePolicy(policyId)
        }
        notifier.deleteNotifier()

        where:
        "Data inputs are"
        legacy << [false, true]
    }

    @Unroll
    @Category([BAT, Notifiers])
    def "Verify Network Simulator Notifications: #type"() {
        when:
        "create notifier"
        for (Notifier notifier : notifierTypes) {
            notifier.createNotifier()
        }

        and:
        "generate a network policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":NOTIFIERDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "send simulation notification"
        withRetry(3, 10) {
            assert NetworkPolicyService.sendSimulationNotification(
                    notifierTypes*.getId(),
                    orchestrator.generateYaml(policy)
            )
        }

        and:
        "validate notification"
        for (Notifier notifier : notifierTypes) {
            notifier.validateNetpolNotification(orchestrator.generateYaml(policy), strictIntegrationTesting)
        }

        cleanup:
        "delete notifiers"
        for (Notifier notifier : notifierTypes) {
            notifier.deleteNotifier()
        }

        where:
        "notifier types"

        type                    | notifierTypes
        "SLACK"                 | [new SlackNotifier()]
        "EMAIL"                 | [new EmailNotifier()]
        "JIRA"                  | [new JiraNotifier()]
        "TEAMS"                 | [new TeamsNotifier()]
        "GENERIC"               | [new GenericNotifier()]

        // Adding a SLACK, TEAMS, EMAIL notifier test so we still verify multiple notifiers
        "SLACK, EMAIL, TEAMS"   | [new SlackNotifier(), new EmailNotifier(), new TeamsNotifier()]
    }

    @Unroll
    @Category([BAT, Notifiers])
    def "Verify Policy Violation Notifications: #type"() {
        when:
        "Create notificaiton(s)"
        for (Notifier notifier : notifierTypes) {
            notifier.createNotifier()
        }

        and:
        "Create policy scoped to test deployment with notification enabled"
        PolicyOuterClass.Policy.Builder policy =
                PolicyOuterClass.Policy.newBuilder(Services.getPolicyByName("Latest tag"))
        policy.setId("")
                .setName("Policy Notifier Test Policy")
                .addScope(ScopeOuterClass.Scope.newBuilder()
                        .setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                                .setKey("app")
                                .setValue(deployment.name)
                        )
                )
        for (Notifier notifier : notifierTypes) {
            policy.addNotifiers(notifier.getId())
        }
        String policyId = CreatePolicyService.createNewPolicy(policy.build())
        assert policyId

        and:
        "create deployment to generate policy violation notification"
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)
        assert Services.waitForViolation(deployment.name, policy.name)

        then:
        "Validate Notification details"
        for (Notifier notifier : notifierTypes) {
            notifier.validateViolationNotification(policy.build(), deployment, strictIntegrationTesting)
        }

        cleanup:
        "delete deployment, policy, and notifiers"
        if (deployment.deploymentUid != null) {
            orchestrator.deleteDeployment(deployment)
        }
        if (policyId != null) {
            CreatePolicyService.deletePolicy(policyId)
        }
        for (Notifier notifier : notifierTypes) {
            notifier.deleteNotifier()
        }

        where:
        "data inputs are:"

        type        | notifierTypes       |
                deployment

        "EMAIL"     | [new EmailNotifier()]       |
                new Deployment()
                        .setName("policy-violation-email-notification")
                        .addLabel("app", "policy-violation-email-notification")
                        .setImage("nginx:latest")

        "PAGERDUTY" | [new PagerDutyNotifier()]   |
                new Deployment()
                        .setName("policy-violation-pagerduty-notification")
                        .addLabel("app", "policy-violation-pagerduty-notification")
                        .setImage("nginx:latest")

        "GENERIC"   | [new GenericNotifier()]     |
                new Deployment()
                        .setName("policy-violation-generic-notification")
                        .addLabel("app", "policy-violation-generic-notification")
                        .setImage("nginx:latest")
    }

    @Unroll
    @Category(Integration)
    def "Verify AWS S3 Integration: #integrationName"() {
        when:
        "the integration is tested"
        def backup = ExternalBackupService.getS3IntegrationConfig(integrationName, bucket, region, endpoint,
                accessKeyId, accesskey)

        then:
        "verify test integration"
        // Test integration for S3 performs test backup (and rollback).
        assert ExternalBackupService.testExternalBackup(backup)

        where:
        "configurations are:"

        integrationName       | bucket                       | region                         |
                endpoint                                             | accessKeyId            |
                accesskey
        "S3 with endpoint"    | Env.mustGetAWSS3BucketName() | Env.mustGetAWSS3BucketRegion() |
                "s3.${Env.mustGetAWSS3BucketRegion()}.amazonaws.com" | Env.mustGetAWSAccessKeyID() |
                Env.mustGetAWSSecretAccessKey()
        "S3 without endpoint" | Env.mustGetAWSS3BucketName() | Env.mustGetAWSS3BucketRegion() |
                ""                                                   | Env.mustGetAWSAccessKeyID() |
                Env.mustGetAWSSecretAccessKey()
        "GCS"                 | Env.mustGetGCSBucketName()   | Env.mustGetGCSBucketRegion()   |
                "storage.googleapis.com"                             | Env.mustGetGCPAccessKeyID() |
                Env.mustGetGCPAccessKey()
    }

    @Unroll
    @Category(Integration)
    def "Verify AWS ECR Integration: #integrationName"() {
        when:
        "the integration is tested"
        def registry = ImageIntegrationService.getECRIntegrationConfig(integrationName, registryID, region, endpoint)

        then:
        "verify test integration"
        assert ImageIntegrationService.testImageIntegration(registry)

        where:
        "configurations are:"

        integrationName        | registryID                    | region                            |
                endpoint
        "ECR with endpoint"    | Env.mustGetAWSECRRegistryID() | Env.mustGetAWSECRRegistryRegion() |
                "ecr.${Env.mustGetAWSECRRegistryRegion()}.amazonaws.com"
        "ECR without endpoint" | Env.mustGetAWSECRRegistryID() | Env.mustGetAWSECRRegistryRegion() |
                ""
    }
}

import groups.Notifiers
import io.fabric8.kubernetes.client.LocalPortForward
import io.grpc.StatusRuntimeException
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.NotifierOuterClass
import groups.BAT
import groups.Integration
import io.stackrox.proto.storage.ScopeOuterClass
import objects.AnchoreScannerIntegration
import objects.AzureRegistryIntegration
import objects.ClairScannerIntegration
import objects.ECRRegistryIntegration
import objects.EmailNotifier
import objects.GCRImageIntegration
import objects.GenericNotifier
import objects.JiraNotifier
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Notifier
import objects.PagerDutyNotifier
import objects.QuayImageIntegration
import objects.SlackNotifier
import objects.SplunkNotifier
import objects.StackroxScannerIntegration
import objects.TeamsNotifier
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
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

    private static final CA_CERT = Env.mustGetInCI("GENERIC_WEBHOOK_SERVER_CA_CONTENTS")

    // https://stack-rox.atlassian.net/browse/ROX-5298 &
    // https://stack-rox.atlassian.net/browse/ROX-5355 &
    // https://stack-rox.atlassian.net/browse/ROX-5789
    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT =
            Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT ? 450 : 30

    def setupSpec() {
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        DEPLOYMENTS.each { Services.waitForDeployment(it) }
    }

    def cleanupSpec() {
        ImageIntegrationService.addStackroxScannerIntegration()
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
                .setImage("stackrox/splunk-test-repo:6.6.1")
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
        try {
            notifier.createNotifier()
        } catch (Exception e) {
            Assume.assumeNoException("Could not create Splunk notifier. Skipping test!", e)
        }

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
        "Create notifications(s)"
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
        assert Services.waitForViolation(deployment.name, policy.name, WAIT_FOR_VIOLATION_TIMEOUT)

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
            notifier.validateViolationResolution()
            notifier.cleanup()
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
    def "Verify #imageIntegration.name() integration - #testAspect"() {
        Assume.assumeTrue(imageIntegration.isTestable())
        Assume.assumeTrue(!testAspect.contains("IAM") || ClusterService.isEKS())

        when:
        "the integration is tested"
        def outcome = ImageIntegrationService.getImageIntegrationClient().testImageIntegration(
                imageIntegration.getCustomBuilder(customArgs).build()
        )

        then:
        "verify test integration outcome"
        assert outcome

        where:
        "tests are:"

        imageIntegration                 | customArgs      | testAspect
        new StackroxScannerIntegration() | [:]             | "default config"
        new AnchoreScannerIntegration()  | [:]             | "default config"
        new ClairScannerIntegration()    | [:]             | "default config"
        new QuayImageIntegration()       | [:]             | "default config"
        new GCRImageIntegration()        | [:]             | "default config"
        new AzureRegistryIntegration()   | [:]             | "default config"
        new ECRRegistryIntegration()     | [:]             | "default config"
        new ECRRegistryIntegration()     | [endpoint: "",] | "without endpoint"
        new ECRRegistryIntegration()     | [useIam: true,] | "requires IAM"
    }

    @Unroll
    @Category(Integration)
    def "Verify improper #imageIntegration.name() integration - #testAspect"() {
        Assume.assumeTrue(imageIntegration.isTestable())

        when:
        "the integration is tested"
        ImageIntegrationService.getImageIntegrationClient().testImageIntegration(
                imageIntegration.getCustomBuilder(getCustomArgs()).build()
        )

        then:
        "verify test integration outcome"
        def error = thrown(expectedError)
        error.message =~ expectedMessage

        where:
        "tests are:"

        imageIntegration                         | getCustomArgs  \
                | expectedError          | expectedMessage      | testAspect

        new StackroxScannerIntegration() | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"

        new AnchoreScannerIntegration() | { [username: Env.mustGet("ANCHORE_USERNAME") + "WRONG",]
        }       | StatusRuntimeException | /401 UNAUTHORIZED/   | "incorrect user"
        new AnchoreScannerIntegration() | { [password: Env.mustGet("ANCHORE_PASSWORD") + "WRONG",]
        }       | StatusRuntimeException | /401 UNAUTHORIZED/   | "incorrect password"
        new AnchoreScannerIntegration() | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"

        new ClairScannerIntegration()   | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"

        new AzureRegistryIntegration() | { [username: "WRONG",]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/   | "incorrect user"
        new AzureRegistryIntegration() | { [password: "WRONG",]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/   | "incorrect password"
        new AzureRegistryIntegration() | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"

        new ECRRegistryIntegration()    | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"

        new ECRRegistryIntegration()    | { [registryId: '0123456789',]
        }       | StatusRuntimeException | /InvalidParameterException/ | "incorrect registry ID"
        new ECRRegistryIntegration()    | { [region: 'nowhere',]
        }       | StatusRuntimeException | /valid region/ | "incorrect region"
        new ECRRegistryIntegration()    | { [accessKeyId: Env.mustGetAWSAccessKeyID() + "OOPS",]
        }       | StatusRuntimeException | /UnrecognizedClientException/ | "incorrect key"
        new ECRRegistryIntegration()    | { [secretAccessKey: Env.mustGetAWSSecretAccessKey() + "OOPS",]
        }       | StatusRuntimeException | /InvalidSignatureException/ | "incorrect secret"

        new QuayImageIntegration()      | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"
        new QuayImageIntegration()      | { [endpoint: "http://169.254.169.254",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference the cluster metadata service/ | "invalid endpoint"
        new QuayImageIntegration()      | { [oauthToken: "EnFzYsRVC4TIBjRenrKt9193KSz9o7vkoWiIGX86",]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/ | "incorrect token"
        new GCRImageIntegration() | { [endpoint: "http://127.0.0.1/nowhere",]
        }       | StatusRuntimeException |
        /invalid endpoint: endpoint cannot reference localhost/ |
        "invalid endpoint"
        new GCRImageIntegration() | { [serviceAccount: Env.mustGet("GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"),]
        }       | StatusRuntimeException | /PermissionDenied/ | "account without access"
        new GCRImageIntegration() | { [project: "not-a-project",]
        }       | StatusRuntimeException | /PermissionDenied/ | "incorrect project"
    }
}

import common.Constants
import groups.BAT
import groups.Integration
import groups.Notifiers
import io.grpc.StatusRuntimeException
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass
import java.util.concurrent.TimeUnit
import objects.AzureRegistryIntegration
import objects.ClairScannerIntegration
import objects.Deployment
import objects.ECRRegistryIntegration
import objects.EmailNotifier
import objects.GCRImageIntegration
import objects.GenericNotifier
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Notifier
import objects.QuayImageIntegration
import objects.SlackNotifier
import objects.SplunkNotifier
import objects.StackroxScannerIntegration
import org.apache.commons.lang3.RandomStringUtils
import org.junit.Assume
import org.junit.AssumptionViolatedException
import org.junit.Rule
import org.junit.experimental.categories.Category
import org.junit.rules.Timeout
import services.ClusterService
import services.ExternalBackupService
import services.ImageIntegrationService
import services.NetworkPolicyService
import services.NotifierService
import services.PolicyService
import spock.lang.Ignore
import spock.lang.Unroll
import util.Env
import util.SplunkUtil

class IntegrationsTest extends BaseSpecification {
    static final private String NOTIFIERDEPLOYMENT = "netpol-notification-test-deployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(NOTIFIERDEPLOYMENT)
                    .setImage("nginx")
                    .addLabel("app", NOTIFIERDEPLOYMENT),
    ]

    private static final CA_CERT = Env.mustGetInCI("GENERIC_WEBHOOK_SERVER_CA_CONTENTS")

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = 30

    @Rule
    @SuppressWarnings(["JUnitPublicProperty"])
    Timeout globalTimeout = new Timeout(1000, TimeUnit.SECONDS)

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
    @Ignore("ROX-8113 - email tests are broken")
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
        "the integration is tested"
        SplunkUtil.SplunkDeployment parts = SplunkUtil.createSplunk(orchestrator,
                Constants.ORCHESTRATOR_NAMESPACE, true)

        when:
        "call the grpc API for the splunk integration."
        SplunkNotifier notifier = new SplunkNotifier(legacy, parts.collectorSvc.name, parts.splunkPortForward.localPort)
        try {
            notifier.createNotifier()
        } catch (Exception e) {
            throw new AssumptionViolatedException("Could not create Splunk notifier. Skipping test!", e)
        }

        and:
        "Edit the policy with the latest keyword."
        PolicyOuterClass.Policy.Builder policy = Services.getPolicyByName("Latest tag").toBuilder()

        def nginxName = "nginx-spl-violation"
        policy.setName("${policy.name} ${parts.uid}")
              .setId("") // set ID to empty so that a new policy is created and not overwrite the original latest tag
              .addScope(ScopeOuterClass.Scope.newBuilder()
                .setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                  .setKey("app")
                  .setValue(nginxName)))
              .addNotifiers(notifier.getId())
        def policyId = PolicyService.createNewPolicy(policy.build())

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
        if (parts.deployment != null) {
            orchestrator.deleteDeployment(nginxdeployment)
        }
        if (policy != null) {
            PolicyService.deletePolicy(policyId)
        }
        if (parts) {
            SplunkUtil.tearDownSplunk(orchestrator, parts)
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
        // ROX-8113 - Email tests are broken
        // "EMAIL"                 | [new EmailNotifier()]
        // "JIRA"                  | [new JiraNotifier()] TODO(ROX-7460)
        // ROX-8145 - Teams tests are broken
        // "TEAMS"                 | [new TeamsNotifier()]
        "GENERIC"               | [new GenericNotifier()]

        // Adding a SLACK, TEAMS, EMAIL notifier test so we still verify multiple notifiers
        // ROX-8113, ROX-8145 - Email and teams tests are broken
        // "SLACK, EMAIL, TEAMS"   | [new SlackNotifier(), new EmailNotifier(), new TeamsNotifier()]

        // Using Slack and Generic to verify multiple notifiers
        "SLACK, GENERIC"        | [new SlackNotifier(), new GenericNotifier()]
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
                                .setValue(deployment.getLabels()["app"])
                        )
                )
        for (Notifier notifier : notifierTypes) {
            policy.addNotifiers(notifier.getId())
        }
        String policyId = PolicyService.createNewPolicy(policy.build())
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
            PolicyService.deletePolicy(policyId)
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

        /*
        // ROX-8113 - Email tests are broken
        "EMAIL"     | [new EmailNotifier()]       |
                new Deployment()
                        // add random id to name to make it easier to search for when validating
                        .setName(uniqueName("policy-violation-email-notification"))
                        .addLabel("app", "policy-violation-email-notification")
                        .setImage("nginx:latest")
        */

        /*
        TODO(ROX-7589)
        "PAGERDUTY" | [new PagerDutyNotifier()]   |
                new Deployment()
                        .setName("policy-violation-pagerduty-notification")
                        .addLabel("app", "policy-violation-pagerduty-notification")
                        .setImage("nginx:latest")
        */
        "GENERIC"   | [new GenericNotifier()]     |
                new Deployment()
                        .setName("policy-violation-generic-notification")
                        .addLabel("app", "policy-violation-generic-notification")
                        .setImage("nginx:latest")
    }

    @Unroll
    @Category([BAT, Notifiers])
    def "Verify Attempted Policy Violation Notifications: #type"() {
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
                                .setValue(deployment.getLabels()["app"])
                        )
                )
                .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
        for (Notifier notifier : notifierTypes) {
            policy.addNotifiers(notifier.getId())
        }
        String policyId = PolicyService.createNewPolicy(policy.build())
        assert policyId

        and:
        "Set admission controller settings to enforce on creates"
        def oldAdmCtrlConfig = ClusterService.getCluster().getDynamicConfig().getAdmissionControllerConfig()

        ClusterOuterClass.AdmissionControllerConfig ac = ClusterOuterClass.AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setTimeoutSeconds(3)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Sleep to allow settings update to propagate
        sleep(5000)

        and:
        "Trigger create deployment to generate attempted policy violation notification"
        def created = orchestrator.createDeploymentNoWait(deployment)

        then:
        "Verify deployment create failed"
        assert !created

        and:
        "Verify attempted alert is generated"
        withRetry(3, 3) {
            def listAlerts = Services.getViolationsWithTimeout(deployment.getName(), "Policy Notifier Test Policy", 60)
            assert listAlerts && listAlerts.get(0).getPolicy().getName() == "Policy Notifier Test Policy"
            // Since the deployment is not created, get the ID from alert.
            def depID = listAlerts.get(0).deployment.id
            assert depID
            deployment.deploymentUid = depID
        }

        and:
        "Validate Notification details"
        for (Notifier notifier : notifierTypes) {
            notifier.validateViolationNotification(policy.build(), deployment, strictIntegrationTesting)
        }

        cleanup:
        "delete deployment, policy, and notifiers"
        if (created) {
            orchestrator.deleteDeployment(deployment)
        }
        if (policyId != null) {
            PolicyService.deletePolicy(policyId)
        }
        for (Notifier notifier : notifierTypes) {
            notifier.cleanup()
            notifier.deleteNotifier()
        }
        ClusterService.updateAdmissionController(oldAdmCtrlConfig)

        where:
        "data inputs are:"

        type        | notifierTypes       |
                deployment

        /*
        ROX-8113 - Email tests are broken
        "EMAIL"     | [new EmailNotifier()]       |
                new Deployment()
                        // add random id to name to make it easier to search for when validating
                        .setName(uniqueName("policy-violation-email-notification"))
                        .addLabel("app", "policy-violation-email-notification")
                        .setImage("nginx:latest")
        */

         /*
         TODO(ROX-7589)
        "PAGERDUTY" | [new PagerDutyNotifier()]   |
                new Deployment()
                        .setName("policy-violation-pagerduty-notification")
                        .addLabel("app", "policy-violation-pagerduty-notification")
                        .setImage("nginx:latest")
        */
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
    @Category([BAT, Notifiers])
    def "Verify Policy Violation Notifications Destination Overrides: #type"() {
        when:
        "Create notifier"
        notifier.createNotifier()
        notifier.notifier

        and:
        "annotate namespace if required"
        if (namespaceAnnotation != null) {
            orchestrator.addNamespaceAnnotation(
                    orchestrator.getNameSpace(),
                    namespaceAnnotation["key"],
                    namespaceAnnotation["value"]
            )
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
                                .setValue(deployment.getLabels()["app"])
                        )
                )
        policy.addNotifiers(notifier.getId())
        String policyId = PolicyService.createNewPolicy(policy.build())
        assert policyId

        and:
        "create deployment to generate policy violation notification"
        orchestrator
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)
        assert Services.waitForViolation(deployment.name, policy.name, WAIT_FOR_VIOLATION_TIMEOUT)

        then:
        "Validate Notification details"
        notifier.validateViolationNotification(policy.build(), deployment, strictIntegrationTesting)

        cleanup:
        "delete deployment, policy, notifiers and clear annotation"
        if (deployment.deploymentUid != null) {
            orchestrator.deleteDeployment(deployment)
        }
        if (policyId != null) {
            PolicyService.deletePolicy(policyId)
        }

        notifier.validateViolationResolution()
        notifier.cleanup()
        notifier.deleteNotifier()

        if (namespaceAnnotation != null) {
            orchestrator.removeNamespaceAnnotation(orchestrator.getNameSpace(), namespaceAnnotation.key)
        }

        where:
        "data inputs are:"

        type     |
                notifier   |
                namespaceAnnotation   |
                deployment

        /*
        // ROX-8113 - Email tests are broken
        "Email deploy override"     |
                new EmailNotifier("Email Test", false,
                        NotifierOuterClass.Email.AuthMethod.DISABLED, null, "stackrox.qa+alt1@gmail.com")   |
                null   |
                new Deployment()
                        // add random id to name to make it easier to search for when validating
                        .setName(uniqueName("policy-violation-email-notification-deploy-override"))
                        .addLabel("app", "policy-violation-email-notification-deploy-override")
                        .addAnnotation("mailgun", "stackrox.qa+alt1@gmail.com")
                        .setImage("nginx:latest")
        "Email namespace override"     |
                new EmailNotifier("Email Test", false,
                        NotifierOuterClass.Email.AuthMethod.DISABLED, null, "stackrox.qa+alt2@gmail.com")   |
                [key: "mailgun", value: "stackrox.qa+alt2@gmail.com"]   |
                new Deployment()
                        // add random id to name to make it easier to search for when validating
                        .setName(uniqueName("policy-violation-email-notification-ns-override"))
                        .addLabel("app", "policy-violation-email-notification-ns-override")
                        .setImage("nginx:latest")
         */
        "Slack deploy override"   |
                new SlackNotifier("slack test", "slack-key")   |
                null   |
                new Deployment()
                        .setName("policy-violation-generic-notification-deploy-override")
                        .addLabel("app", "policy-violation-generic-notification-deploy-override")
                        .addAnnotation("slack-key", NotifierService.SLACK_ALT_WEBHOOK)
                        .setImage("nginx:latest")
        "Slack namespace override"   |
                new SlackNotifier("slack test", "slack-key")   |
                [key: "slack-key", value: NotifierService.SLACK_ALT_WEBHOOK] |
                new Deployment()
                        .setName("policy-violation-generic-notification-ns-override")
                        .addLabel("app", "policy-violation-generic-notification-ns-override")
                        .setImage("nginx:latest")
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
        }       | StatusRuntimeException | /INVALID_ARGUMENT/ | "incorrect registry ID"
        new ECRRegistryIntegration()    | { [region: 'nowhere',]
        }       | StatusRuntimeException | /valid region/ | "incorrect region"
        new ECRRegistryIntegration()    | { [accessKeyId: Env.mustGetAWSAccessKeyID() + "OOPS",]
        }       | StatusRuntimeException | /UnrecognizedClientException/ | "incorrect key"
        new ECRRegistryIntegration()    | { [secretAccessKey: Env.mustGetAWSSecretAccessKey() + "OOPS",]
        }       | StatusRuntimeException | /InvalidSignatureException/ | "incorrect secret"

        new ECRRegistryIntegration()    | { [useAssumeRole: true,]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/ | "AssumeRole with endpoint set"
        new ECRRegistryIntegration()    | { [useAssumeRole: true, assumeRoleRoleId: "OOPS", endpoint: "",]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/ | "AssumeRole with incorrect role"
        new ECRRegistryIntegration()    | { [useAssumeRoleExternalId: true, assumeRoleExternalId: "OOPS", endpoint: "",]
        }       | StatusRuntimeException | /INVALID_ARGUMENT/ | "AssumeRole external ID with incorrect external ID"

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

    // TODO disabled due to flaking (ROX-7902), shoudl be re-enabled once reworked (ROX-10310)
    //@Category(Integration)
    //def "Verify syslog notifier"() {
    //    given:
    //    "the some syslog receiver is created"
    //    // Change the local port numbers so we don't conflict with any other splunk instances
    //    SplunkUtil.SplunkDeployment splunkDeployment = SplunkUtil.createSplunk(orchestrator,
    //            Constants.ORCHESTRATOR_NAMESPACE, true)

    //    when:
    //    "call the grpc API for the syslog integration."
    //    SyslogNotifier notifier = new SyslogNotifier(splunkDeployment.syslogSvc.name, 514,
    //            splunkDeployment.splunkPortForward.localPort)
    //    try {
    //        notifier.createNotifier()
    //    } catch (Exception e) {
    //        throw new AssumptionViolatedException("Could not create syslog notifier. Skipping test!", e)
    //    }

    //    then:
    //    "Verify the messages are seen in the json"
    //    // We should have at least one audit log for the message which created the syslog integration.
    //    notifier.validateViolationNotification(null, null, false)

    //    cleanup:
    //    "remove splunk and syslog notifier integration"
    //    SplunkUtil.tearDownSplunk(orchestrator, splunkDeployment)
    //    notifier.deleteNotifier()
    //}

    def uniqueName(String name) {
        return name + RandomStringUtils.randomAlphanumeric(5).toLowerCase()
    }
}

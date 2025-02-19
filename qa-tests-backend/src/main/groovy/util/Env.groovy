package util

import groovy.transform.CompileStatic
import orchestratormanager.OrchestratorTypes

import java.nio.file.Files
import java.nio.file.Paths
import java.nio.file.attribute.FileTime
import org.slf4j.Logger
import org.slf4j.LoggerFactory

@CompileStatic
class Env {

    private static final Logger LOG = LoggerFactory.getLogger(this.getClass())

    private static final String PROPERTIES_FILE = "qa-test-settings.properties"

    private static final Map<String, String> DEFAULT_VALUES = [
            "API_HOSTNAME": "localhost",
            "API_PORT": "8000",
            "ROX_USERNAME": "admin",
    ]

    static final boolean IN_CI = (System.getenv("CI") == "true")
    static final String CI_JOB_NAME = System.getenv("CI_JOB_NAME")
    static final String BUILD_TAG = System.getenv("BUILD_TAG")
    static final boolean GATHER_QA_TEST_DEBUG_LOGS = (System.getenv("GATHER_QA_TEST_DEBUG_LOGS") == "true")
    static final String QA_TEST_DEBUG_LOGS = System.getenv("QA_TEST_DEBUG_LOGS") ?: ""
    static final boolean HAS_WORKLOAD_IDENTITIES = (System.getenv("SETUP_WORKLOAD_IDENTITIES") == "true")

    static final String IMAGE_PULL_POLICY_FOR_QUAY_IO = System.getenv("IMAGE_PULL_POLICY_FOR_QUAY_IO")

    // REMOTE_CLUSTER_ARCH specifies architecture of a remote cluster on which tests are to be executed
    // the remote cluster arch can be ppc64le or s390x, default is x86_64
    static final boolean REMOTE_CLUSTER_ARCH = System.getenv("REMOTE_CLUSTER_ARCH") ?: "x86_64"

    // ONLY_SECURED_CLUSTER specifies that the remote cluster being used to execute tests
    // only has secured-cluster deployed and connects to a remote central
    static final ONLY_SECURED_CLUSTER = System.getenv("ONLY_SECURED_CLUSTER") ?: "false"

    private static final Env INSTANCE = new Env()

    static String get(String key, String defVal = null) {
        return INSTANCE.getInternal(key, defVal)
    }

    static String mustGet(String key) {
        return INSTANCE.mustGetInternal(key)
    }

    static String mustGetInCI(String key, String defVal = null) {
        return INSTANCE.mustGetInCIInternal(key, defVal)
    }

    private final Properties envVars = new Properties()

    private Env() {
        if (!IN_CI) {
            loadEnvVarsFromPropsFile()
        }
        envVars.putAll(System.getenv())
        if (!IN_CI) {
            assignFallbackValues()
        }
    }

    private loadEnvVarsFromPropsFile() {
        try {
            envVars.load(new FileInputStream(PROPERTIES_FILE))
        } catch (Exception ex) {
            LOG.error( "Failed to load extra properties file", ex)
        }
    }

    protected String getInternal(String key, String defVal) {
        return envVars.getOrDefault(key, defVal)
    }

    protected String mustGetInternal(String key) {
        def value = envVars.get(key)
        if (value == null) {
            throw new RuntimeException("No value assigned for required key ${key}")
        }
        return value
    }

    protected String mustGetInCIInternal(String key, String defVal) {
        def value = envVars.get(key)
        if (value == null) {
            if (IN_CI) {
                throw new RuntimeException("No value assigned for required key ${key}")
            }
            return defVal
        }
        return value
    }

    protected boolean isEnvVarEmpty(String key) {
        return (envVars.get(key) ?: "null") == "null"
    }

    private static OrchestratorTypes inferOrchestratorType() {
        // Infer the orchestrator type from the local deployment, by looking at which
        // `deploy/<orchestrator>/central-deploy/password` file was most recently written to.
        OrchestratorTypes selected = null
        FileTime mostRecent = null
        for (def orchestratorType : OrchestratorTypes.values()) {
            def passwordPath = "../deploy/${orchestratorType.toString().toLowerCase()}" +
                    "/central-deploy/password"
            try {
                def modTime = Files.getLastModifiedTime(Paths.get(passwordPath))
                if (mostRecent == null || modTime > mostRecent) {
                    selected = orchestratorType
                    mostRecent = modTime
                }
            } catch (Exception ex) {
                LOG.debug("error inferOrchestratorType", ex) // no-op
            }
        }

        return selected ?: OrchestratorTypes.K8S
    }

    private void assignFallbackValues() {
        for (def entry : DEFAULT_VALUES.entrySet()) {
            if (isEnvVarEmpty(entry.key)) {
                envVars.put(entry.key, entry.value)
            }
        }
        LOG.debug System.getenv().toMapString()

        if (isEnvVarEmpty("ROX_ADMIN_PASSWORD")) {
            if (isEnvVarEmpty("CLUSTER")) {
                envVars.put("CLUSTER", inferOrchestratorType().toString())
            }

            String password = null
            try {
                def passwordPath = "../deploy/${envVars.get("CLUSTER").toString().toLowerCase()}/central-deploy/password"
                BufferedReader br = new BufferedReader(new FileReader(passwordPath))
                password = br.readLine()
            } catch (Exception ex) {
                LOG.warn("Failed to load password for current deployment", ex)
            }

            if (password != null) {
                envVars.put("ROX_ADMIN_PASSWORD", password)
            }
        }

        if (isEnvVarEmpty("CLUSTER")) {
            envVars.put("CLUSTER", OrchestratorTypes.K8S.toString())
        }
    }

    static String mustGetUsername() {
        return mustGet("ROX_USERNAME")
    }

    static String mustGetPassword() {
        return mustGet("ROX_ADMIN_PASSWORD")
    }

    static int mustGetPort() {
        String portString = mustGet("API_PORT")
        int port
        try {
            port = Integer.parseInt(portString)
        } catch (NumberFormatException e) {
            throw new RuntimeException("API_PORT " + portString + " is not a valid number " + e.toString())
        }
        return port
    }

    static String mustGetHostname() {
        return mustGet("API_HOSTNAME")
    }

    static OrchestratorTypes mustGetOrchestratorType() {
        String cluster = mustGet("CLUSTER")
        OrchestratorTypes type
        try {
            type = OrchestratorTypes.valueOf(cluster)
        } catch (IllegalArgumentException e) {
            throw new RuntimeException("CLUSTER must be one of " + OrchestratorTypes.values() + " " + e.toString())
        }
        return type
    }

    static String mustGetKeystorePath() {
        return mustGet("KEYSTORE_PATH")
    }

    static String mustGetTruststorePath() {
        return mustGet("TRUSTSTORE_PATH")
    }

    static String mustGetClientCAPath() {
        return mustGet("CLIENT_CA_PATH")
    }

    static String mustGetImageTag() {
        return mustGet("IMAGE_TAG")
    }

    static String mustGetResultsFilePath() {
        return mustGet("RESULTS_FILE_PATH")
    }

    static String mustGetAWSAccessKeyID() {
        return mustGet("AWS_ACCESS_KEY_ID")
    }

    static String mustGetAWSSecretAccessKey() {
        return mustGet("AWS_SECRET_ACCESS_KEY")
    }

    static String mustGetAWSAssumeRoleAccessKeyID() {
        return mustGet("AWS_ASSUME_ROLE_ACCESS_KEY_ID")
    }

    static String mustGetAWSAssumeRoleSecretKeyID() {
        return mustGet("AWS_ASSUME_ROLE_SECRET_ACCESS_KEY")
    }

    static String mustGetAWSAssumeRoleRoleID() {
        return mustGet("AWS_ASSUME_ROLE_ROLE_ID")
    }

    static String mustGetAWSAssumeRoleExternalID() {
        return mustGet("AWS_ASSUME_ROLE_EXTERNAL_ID")
    }

    static String mustGetAWSAssumeRoleTestConditionID() {
        return mustGet("AWS_ASSUME_ROLE_TEST_CONDITION_ID")
    }

    static String mustGetAWSS3BucketName() {
        return mustGet("AWS_S3_BACKUP_TEST_BUCKET_NAME") // stackrox-qa-backup-test
    }

    static String mustGetAWSS3BucketRegion() {
        return mustGet("AWS_S3_BACKUP_TEST_BUCKET_REGION") // us-east-2
    }

    static String mustGetAWSECRRegistryID() {
        return mustGet("AWS_ECR_REGISTRY_NAME") // 051999192406
    }

    static String mustGetAWSECRRegistryRegion() {
        return mustGet("AWS_ECR_REGISTRY_REGION") // us-east-2
    }

    static String mustGetAWSECRDockerPullPassword() {
        return mustGet("AWS_ECR_DOCKER_PULL_PASSWORD") // aws ecr get-login-password
    }

    static String mustGetCloudflareR2BucketName() {
        return mustGet("CLOUDFLARE_R2_BACKUP_TEST_BUCKET_NAME") // stackrox-ci-qa-backup-test
    }

    static String mustGetCloudflareR2BucketRegion() {
        return mustGet("CLOUDFLARE_R2_BACKUP_TEST_REGION") // ENAM
    }

    static String mustGetCloudflareR2Endpoint() {
        return "${mustGet("CLOUDFLARE_R2_BACKUP_TEST_ACCOUNT_ID")}.r2.cloudflarestorage.com"
    }

    static String mustGetCloudflareR2AccessKeyID() {
        return mustGet("CLOUDFLARE_R2_BACKUP_TEST_ACCESS_KEY_ID")
    }

    static String mustGetCloudflareR2SecretAccessKey() {
        return mustGet("CLOUDFLARE_R2_BACKUP_TEST_SECRET_ACCESS_KEY")
    }

    static String mustGetGCSBucketName() {
        return mustGet("GCP_GCS_BACKUP_TEST_BUCKET_NAME_V2")
    }

    static String mustGetGCPAccessKeyID() {
        return mustGet("GCP_ACCESS_KEY_ID_V2")
    }

    static String mustGetGCPAccessKey() {
        return mustGet("GCP_SECRET_ACCESS_KEY_V2")
    }

    static String mustGetGCSServiceAccount() {
        return mustGet("GOOGLE_GCS_BACKUP_SERVICE_ACCOUNT_V2")
    }

    static String mustGetGCRServiceAccount() {
        return mustGet("GOOGLE_CREDENTIALS_GCR_SCANNER_V2")
    }

    static String mustGetGCRNoAccessServiceAccount() {
        return mustGet("GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY_V2")
    }

    static String mustGetPagerdutyToken() {
        return mustGet("PAGERDUTY_TOKEN")
    }

    static String mustGetSlackFixableVulnsChannel() {
        return mustGet("SLACK_FIXABLE_VULNS_CHANNEL")
    }

    static String mustGetSlackMainWebhook() {
        return mustGet("SLACK_MAIN_WEBHOOK")
    }

    static String mustGetSlackAltWebhook() {
        return mustGet("SLACK_ALT_WEBHOOK")
    }

    static String getDisableAuditLogAlertsTest() {
        return get("DISABLE_AUDIT_LOG_ALERTS_TEST")
    }

    static String getManagedControlPlane() {
        return get("MANAGED_CP", "false")
    }

    static String getSupportsLoadBalancerSvc() {
        return get("SUPPORTS_LOAD_BALANCER_SVC", "true")
    }

    static String mustGetOcmOfflineToken() {
        return get("OCM_OFFLINE_TOKEN")
    }

    static String mustGetOcmClientId() {
        return get("CLOUD_SOURCES_TEST_OCM_CLIENT_ID")
    }

    static String mustGetOcmClientSecret() {
        return get("CLOUD_SOURCES_TEST_OCM_CLIENT_SECRET")
    }

    static String getTestTarget() {
        return get("TEST_TARGET", "")
    }
}

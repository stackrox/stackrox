import static Services.waitForViolation

import groups.BAT
import objects.ConfigMapKeyRef
import objects.Deployment
import objects.SecretKeyRef
import objects.Volume
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicySection
import io.stackrox.proto.storage.PolicyOuterClass.PolicyValue
import services.AlertService
import services.CreatePolicyService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env

class PolicyFieldsTest extends BaseSpecification {
    static final private Deployment DEP_A =
            new Deployment()
                .setName("deployment-a")
                .setImage("us.gcr.io/stackrox-ci/qa/trigger-policy-violations/more:0.3")
                .setCapabilities(["NET_ADMIN", "SYSLOG"], ["IPC_LOCK", "WAKE_ALARM"])
                .addLimits("cpu", "0.5")
                .addRequest("cpu", "0.25")
                .addLimits("memory", "500Mi")
                .addRequest("memory", "250Mi")
                .addAnnotation("im-a-key", "")
                .addLabel("im-a-key", "")
                .setEnv(["ENV_FROM_RAW": "VALUE"])
                .addEnvValueFromConfigMapKeyRef(
                        "ENV_FROM_CONFIG_MAP_KEY",
                        new ConfigMapKeyRef(name: CONFIG_MAP_NAME, key: "some_configuration"))
                .addEnvValueFromSecretKeyRef(
                        "ENV_FROM_SECRET_KEY",
                        new SecretKeyRef(key: "password", name: SECRET_NAME))
                .addEnvValueFromFieldRef("ENV_FROM_FIELD", "metadata.name")
                .addEnvValueFromResourceFieldRef("ENV_FROM_RESOURCE_FIELD", "limits.cpu")
                .addPort(25, "TCP")
                .setCreateLoadBalancer(true)
                .setExposeAsService(true)
                .setPrivilegedFlag(true)
                .addVolume(
                        new Volume(
                                name: "foo-volume",
                                hostPath: false,
                                mountPath: "/tmp/foo-volume"
                        )
                )

    static final private BASED_ON_DEBIAN_7 = DEP_A
    static final private WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG = DEP_A
    static final private WITHOUT_CVE_2019_5436 = DEP_A
    static final private WITH_CVSS_LT_8 = DEP_A
    static final private WITH_CPU_LIMIT_HALF = DEP_A
    static final private WITH_CPU_REQUEST_QUARTER = DEP_A
    static final private WITH_MEMORY_LIMIT_500MI = DEP_A
    static final private WITH_MEMORY_REQUEST_250MI = DEP_A
    static final private WITH_KEY_ONLY_ANNOTATION = DEP_A
    static final private WITH_KEY_ONLY_LABEL = DEP_A
    static final private WITH_IMAGE_LABELS = DEP_A
    static final private WITH_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM = DEP_A
    static final private WITH_RAW_ENV_AND_VALUE = DEP_A
    static final private WITH_ENV_FROM_CONFIG_MAP_KEY = DEP_A
    static final private WITH_ENV_FROM_SECRET_KEY = DEP_A
    static final private WITH_ENV_FROM_FIELD = DEP_A
    static final private WITH_ENV_FROM_RESOURCE_FIELD = DEP_A
    static final private USES_USGCR = DEP_A
    static final private WITH_IMAGE_REMOTE_TO_MATCH = DEP_A
    static final private WITH_IMAGE_TAG_TO_NOT_MATCH = DEP_A
    static final private WITH_RECENT_SCAN_AGE = DEP_A
    static final private WITH_PORT_25_EXPOSED = DEP_A
    static final private WITH_LB_SERVICE = DEP_A
    static final private WITH_PRIVILEGE = DEP_A
    static final private WITH_PROCESS_UID_1 = DEP_A
    static final private WITH_RW_ROOT_FS = DEP_A
    static final private IS_SCANNED = DEP_A
    static final private WITH_A_RW_FOO_VOLUME = DEP_A

    static final private Deployment DEP_B =
            new Deployment()
                .setName("deployment-b")
                .setImage("us.gcr.io/stackrox-ci/qa/trigger-policy-violations/most:0.19")
                .setCapabilities(["NET_ADMIN"], ["IPC_LOCK"])
                .addLimits("cpu", "1")
                .addRequest("cpu", "0.5")
                .addLimits("memory", "1Gi")
                .addRequest("memory", "0.5Gi")
                .addAnnotation("im-a-key", "and a value")
                .addLabel("im-a-key", "and_a_value")
                .setEnv(["ENV_FROM_RAW": "VALUE DIFFERENT"])
                .addEnvValueFromConfigMapKeyRef(
                        "DIFFERENT_ENV_FROM_CONFIG_MAP_KEY",
                        new ConfigMapKeyRef(name: CONFIG_MAP_NAME, key: "some_configuration"))
                .addEnvValueFromSecretKeyRef(
                        "DIFFERENT_ENV_FROM_SECRET_KEY",
                        new SecretKeyRef(key: "password", name: SECRET_NAME))
                .addEnvValueFromFieldRef("DIFFERENT_ENV_FROM_FIELD", "metadata.name")
                .addEnvValueFromResourceFieldRef("DIFFERENT_ENV_FROM_RESOURCE_FIELD", "limits.cpu")
                .setPrivilegedFlag(false)
                .addVolume(
                        new Volume(
                                name: "bar-volume",
                                hostPath: true,
                                mountPath: "/tmp"
                        ),
                        true
                )

    static final private BASED_ON_CENTOS_8 = DEP_B
    static final private WITH_ADD_CAPS_NET_ADMIN = DEP_B
    static final private WITH_CVE_2019_5436 = DEP_B
    static final private WITH_CVSS_GT_8 = DEP_B
    static final private WITH_CPU_LIMIT_ONE = DEP_B
    static final private WITH_CPU_REQUEST_HALF = DEP_B
    static final private WITH_MEMORY_LIMIT_ONEGI = DEP_B
    static final private WITH_MEMORY_REQUEST_HALFGI = DEP_B
    static final private WITH_KEY_AND_VALUE_ANNOTATION = DEP_B
    static final private WITH_KEY_AND_VALUE_LABEL = DEP_B
    static final private WITHOUT_IMAGE_LABELS = DEP_B
    static final private WITH_DROP_CAPS_IPC_LOCK = DEP_B
    static final private WITH_RAW_ENV_AND_DIFFERENT_VALUE = DEP_B
    static final private WITH_DIFFERENT_ENV_FROM_CONFIG_MAP_KEY = DEP_B
    static final private WITH_DIFFERENT_ENV_FROM_SECRET_KEY = DEP_B
    static final private WITH_DIFFERENT_ENV_FROM_FIELD = DEP_B
    static final private WITH_DIFFERENT_ENV_FROM_RESOURCE_FIELD = DEP_B
    static final private OLDER_THAN_1_DAY = DEP_B
    static final private WITH_IMAGE_REMOTE_TO_NOT_MATCH = DEP_B
    static final private WITH_IMAGE_TAG_TO_MATCH = DEP_B
    static final private WITHOUT_PORTS_EXPOSED = DEP_B
    static final private WITHOUT_SERVICE = DEP_B
    static final private WITHOUT_PRIVILEGE = DEP_B
    static final private WITH_BASH_PARENT = DEP_B
    static final private WITH_VERSION_ARG = DEP_B
    static final private WITH_BASH_EXEC = DEP_B
    static final private WITHOUT_PROCESS_UID_1 = DEP_B
    static final private WITH_A_RO_HOST_BAR_VOLUME = DEP_B

    static final private CONFIG_MAP_NAME = "test-config-map"
    static final private SECRET_NAME = "test-secret"

    static final private Deployment DEP_C =
            new Deployment()
                    .setName("deployment-c")
                    .setImage("us.gcr.io/stackrox-ci/qa/trigger-policy-violations/alpine:0.6")
                    .addAnnotation("im-a-key", "with a different value")
                    .addAnnotation("another-key", "and a value")
                    .addLabel("im-a-key", "with_a_different_value")
                    .addLabel("another-key", "and_a_value")
                    .setReadOnlyRootFilesystem(true)

    static final private BASED_ON_ALPINE = DEP_C
    static final private WITHOUT_ADD_CAPS = DEP_C
    static final private WITHOUT_CPU_LIMIT = DEP_C
    static final private WITHOUT_CPU_REQUEST = DEP_C
    static final private WITHOUT_MEMORY_LIMIT = DEP_C
    static final private WITHOUT_MEMORY_REQUEST = DEP_C
    static final private WITH_MISMATCHED_ANNOTATIONS = DEP_C
    static final private WITH_MISMATCHED_LABELS = DEP_C
    static final private WITHOUT_DROP_CAPS = DEP_C
    static final private WITHOUT_ENV = DEP_C
    static final private YOUNGER_THAN_TEN_YEARS = DEP_C
    static final private WITHOUT_COMPONENT_CPIO = DEP_C
    static final private WITHOUT_BASH_PARENT = DEP_C
    static final private WITHOUT_BASH_EXEC = DEP_C
    static final private WITH_RDONLY_ROOT_FS = DEP_C
    static final private WITHOUT_FOO_OR_BAR_VOLUMES = DEP_C

    static final private Deployment DEP_D =
        new Deployment()
        .setName("deployment-d")
        .setImage("docker.io/stackrox/qa:apache-dns")

    static final private WITHOUT_ANNOTATIONS = DEP_D
    static final private WITH_COMPONENT_CPIO = DEP_D
    static final private WITHOUT_VERSION_ARG = DEP_D
    static final private USES_DOCKER = DEP_D

    static final private Deployment DEP_E =
            new Deployment()
                    .setName("deployment-e")
                    .setImage("non-existent:image")

    static final private UNSCANNED = DEP_E

    static final private SENSOR = new Deployment()
            .setName("sensor")
            .setNamespace("stackrox")

    static final private CENTRAL = new Deployment()
            .setName("central")
            .setNamespace("stackrox")

    static final private List<Deployment> DEPLOYMENTS = [
            DEP_A,
            DEP_B,
            DEP_C,
            DEP_D,
    ]

    static final private Map<String, String> CONFIG_MAP_DATA = [
            "some_configuration": "a value",
    ]

    // https://stack-rox.atlassian.net/browse/ROX-5298
    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT =
            Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT ? 100 : 30

    static final private BASE_POLICY = Policy.newBuilder()
            .addLifecycleStages(LifecycleStage.DEPLOY)
            .addCategories("Test")
            .setDisabled(false)
            .setSeverityValue(2)

    static final private BASE_RUNTIME_POLICY = BASE_POLICY.clone()
            .clearLifecycleStages()
            .addLifecycleStages(LifecycleStage.RUNTIME)

    // "Add Capabilities"

    static final private NO_ADD_CAPS_NET_ADMIN = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_ADD_CAPS_NET_ADMIN"),
            "Add Capabilities",
            ["NET_ADMIN"]
    )

    static final private NO_ADD_CAPS_SYSLOG = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_ADD_CAPS_SYSLOG"),
            "Add Capabilities",
            ["SYSLOG"]
    )

    static final private NO_ADD_CAPS_NET_ADMIN_AND_SYSLOG = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_ADD_CAPS_NET_ADMIN_AND_SYSLOG"),
            "Add Capabilities",
            ["NET_ADMIN", "SYSLOG"]
    )

    static final private NO_ADD_CAPS_LEASE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_ADD_CAPS_LEASE"),
            "Add Capabilities",
            ["LEASE"]
    )

    static final private NO_ADD_CAPS_NET_ADMIN_AND_LEASE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_ADD_CAPS_NET_ADMIN_AND_LEASE"),
            "Add Capabilities",
            ["NET_ADMIN", "LEASE"]
    )

    // "CVE"

    static final private EXCLUDE_CVE_2019_5436 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_EXCLUDE_CVE_2019_5436"),
            "CVE",
            ["CVE-2019-5436"]
    )

    // "CVSS"

    static final private EXCLUDE_CVSS_GT_8 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_EXCLUDE_CVSS_GT_8"),
            "CVSS",
            ["> 8"]
    )

    // "Container CPU Limit"

    static final private CPU_LIMIT_GT_0PT7 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_CPU_LIMIT_GT_0PT7"),
            "Container CPU Limit",
            ["> 0.7"]
    )

    static final private CPU_LIMIT_GE_1 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_CPU_LIMIT_GE_1"),
            "Container CPU Limit",
            [">= 1"]
    )

    // "Container CPU Request"

    static final private CPU_REQUEST_LT_HALF = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_CPU_REQUEST_LT_HALF"),
            "Container CPU Request",
            ["< 0.5"]
    )

    static final private CPU_REQUEST_GT_HALF = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_CPU_REQUEST_GT_HALF"),
            "Container CPU Request",
            ["> 0.5"]
    )

    // "Container Memory Limit"

    static final private MEMORY_LIMIT_LE_750MI = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_MEMORY_LIMIT_LE_750MI"),
            "Container Memory Limit",
            ["<= 750"]
    )

    static final private MEMORY_LIMIT_GE_750MI = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_MEMORY_LIMIT_GE_750MI"),
            "Container Memory Limit",
            [">= 750"]
    )

    // "Container Memory Request"

    static final private MEMORY_REQUEST_EQ_250MI = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_MEMORY_REQUEST_EQ_250MI"),
            "Container Memory Request",
            ["250"]
    )

    // "Disallowed Annotation"

    static final private DISALLOWED_ANNOTATION_KEY_ONLY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_ANNOTATION_KEY_ONLY"),
            "Disallowed Annotation",
            ["im-a-key="]
    )

    static final private DISALLOWED_ANNOTATION_KEY_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_ANNOTATION_KEY_AND_VALUE"),
            "Disallowed Annotation",
            ["im-a-key=and a value"]
    )

    // "Disallowed Image Label"

    static final private DISALLOWED_IMAGE_LABEL_KEY_ONLY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_IMAGE_LABEL_KEY_ONLY"),
            "Disallowed Image Label",
            ["test.com-i-am-a-key="]
    )

    static final private DISALLOWED_IMAGE_LABEL_KEY_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_IMAGE_LABEL_KEY_AND_VALUE"),
            "Disallowed Image Label",
            ["test.com-i-am-another-key=another value"]
    )

    static final private DISALLOWED_IMAGE_LABEL_NO_MATCH_I = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_IMAGE_LABEL_NO_MATCH_I"),
            "Disallowed Image Label",
            ["no.match-key="]
    )

    static final private DISALLOWED_IMAGE_LABEL_NO_MATCH_II = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_DISALLOWED_IMAGE_LABEL_NO_MATCH_II"),
            "Disallowed Image Label",
            ["no.match-key=a value"]
    )

    // "Drop Capabilities"

    static final private HAS_DROP_CAPS_IPC_LOCK = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_DROP_CAPS_IPC_LOCK_X"),
            "Drop Capabilities",
            ["IPC_LOCK"]
    )

    static final private HAS_DROP_CAPS_WAKE_ALARM = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_DROP_CAPS_WAKE_ALARM"),
            "Drop Capabilities",
            ["WAKE_ALARM"]
    )

    static final private HAS_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM"),
            "Drop Capabilities",
            ["IPC_LOCK", "WAKE_ALARM"]
    )

    static final private HAS_DROP_CAPS_LEASE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_DROP_CAPS_LEASE"),
            "Drop Capabilities",
            ["LEASE"]
    )

    static final private HAS_DROP_CAPS_IPC_LOCK_AND_LEASE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_DROP_CAPS_IPC_LOCK_AND_LEASE"),
            "Drop Capabilities",
            ["IPC_LOCK", "LEASE"]
    )

    // "Environment Variable"

    static final private HAS_RAW_ENV = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_RAW_ENV"),
            "Environment Variable",
            ["RAW=ENV_FROM_RAW="]
    )

    static final private HAS_RAW_ENV_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_RAW_ENV_AND_VALUE"),
            "Environment Variable",
            ["RAW=ENV_FROM_RAW=VALUE"]
    )

    static final private HAS_ENV_FROM_CONFIG_MAP_KEY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_ENV_FROM_CONFIG_MAP_KEY"),
            "Environment Variable",
            ["CONFIG_MAP_KEY=ENV_FROM_CONFIG_MAP_KEY="] // Note: values are not followed into the config map
                                                        // and nor are they ignored ROX-5208
    )

    static final private HAS_ENV_FROM_SECRET_KEY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_ENV_FROM_SECRET_KEY"),
            "Environment Variable",
            ["SECRET_KEY=ENV_FROM_SECRET_KEY="]
    )

    static final private HAS_ENV_FROM_FIELD = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_ENV_FROM_FIELD"),
            "Environment Variable",
            ["FIELD=ENV_FROM_FIELD="]
    )

    static final private HAS_ENV_FROM_RESOURCE_FIELD = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_ENV_FROM_RESOURCE_FIELD"),
            "Environment Variable",
            ["RESOURCE_FIELD=ENV_FROM_RESOURCE_FIELD="]
    )

    // "Image Age"

    static final private IS_GREATER_THAN_1_DAY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_IS_GREATER_THAN_1_DAY"),
            "Image Age",
            ["1"]
    )

    static final private IS_GREATER_THAN_10_YEARS = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_IS_GREATER_THAN_10_YEARS"),
            "Image Age",
            ["3650"]
    )

    // "Image Component"

    static final private HAS_COMPONENT_CPIO = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_COMPONENT_CPIO"),
            "Image Component",
            ["cpio="]
    )

    static final private HAS_COMPONENT_CPIO_WITH_VERSION = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_COMPONENT_CPIO_WITH_VERSION"),
            "Image Component",
            ["cpio=2.11\\+dfsg\\-1ubuntu1.2"]
    )

    static final private HAS_COMPONENT_CPIO_WITH_OTHER_VERSION = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_COMPONENT_CPIO_WITH_OTHER_VERSION"),
            "Image Component",
            ["cpio=2.12\\+dfsg\\-1ubuntu1.2"]
    )

    // "Image OS"

    static final private IS_BASED_ON_DEBIAN_7 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_BASED_ON_DEBIAN_7"),
            "Image OS",
            ["debian:7"]
    )

    static final private IS_BASED_ON_CENTOS_8 = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_BASED_ON_CENTOS_8"),
            "Image OS",
            ["centos:8"]
    )

    static final private IS_BASED_ON_ALPINE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_BASED_ON_ALPINE"),
            "Image OS",
            ["alpine"]
    )

    // "Image Registry"

    static final private NO_IMAGE_REGISTRY_USCGR = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_IMAGE_REGISTRY_USCGR"),
            "Image Registry",
            ["us.gcr.io"]
    )

    // "Image Remote"

    static final private NO_IMAGE_REMOTE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_IMAGE_REMOTE"),
            "Image Remote",
            ["stackrox-ci/qa/trigger-policy-violations/more"]
    )

    // "Image Tag"

    static final private NO_IMAGE_TAG = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_IMAGE_TAG"),
            "Image Tag",
            ["0.19"]
    )

    // "Image Scan Age"

    static final private NO_OLD_IMAGE_SCANS = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_OLD_IMAGE_SCANS"),
            "Image Scan Age",
            ["30"]
    )

    // "Minimum RBAC Permissions"

    static final private MINIMUM_RBAC_CLUSTER_WIDE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_MINIMUM_RBAC_CLUSTER_WIDE"),
            "Minimum RBAC Permissions",
            ["ELEVATED_CLUSTER_WIDE"]
    )

    // "Exposed Port"

    static final private HAS_PORT_25_EXPOSED = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_PORT_25_EXPOSED"),
            "Exposed Port",
            ["25"]
    )

    // "Port Exposure Method"

    static final private HAS_EXTERNAL_EXPOSURE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_EXTERNAL_EXPOSURE"),
            "Port Exposure Method",
            ["EXTERNAL"]
    )

    // "Privileged Container"

    static final private IS_PRIVILEGED = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_IS_PRIVILEGED"),
            "Privileged Container",
            ["true"]
    )

    // "Process Ancestor"

    static final private HAS_BASH_PARENT = setPolicyFieldANDValues(
            BASE_RUNTIME_POLICY.clone().setName("AAA_HAS_BASH_PARENT"),
            "Process Ancestor",
            ["/bin/bash"]
    )

    // "Process Arguments"

    static final private HAS_VERSION_ARGS = setPolicyFieldANDValues(
            BASE_RUNTIME_POLICY.clone().setName("AAA_HAS_VERSION_ARGS"),
            "Process Arguments",
            ["--version"]
    )

    // "Process Name"

    static final private HAS_BASH_EXEC = setPolicyFieldANDValues(
            BASE_RUNTIME_POLICY.clone().setName("AAA_HAS_BASH_EXEC"),
            "Process Name",
            [".*bash"]
    )

    // "Process UID"

    static final private HAS_PROCESS_UID_1 = setPolicyFieldANDValues(
            BASE_RUNTIME_POLICY.clone().setName("AAA_HAS_PROCESS_UID_1"),
            "Process UID",
            ["1"]
    )

    // "Read-Only Root Filesystem"

    static final private HAS_RW_ROOT_FS = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_HAS_RW_ROOT_FS"),
            "Read-Only Root Filesystem",
            ["false"]
    )

    // "Required Annotation"

    static final private REQUIRED_ANNOTATION_KEY_ONLY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_ANNOTATION_KEY_ONLY"),
            "Required Annotation",
            ["im-a-key="]
    )

    static final private REQUIRED_ANNOTATION_KEY_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_ANNOTATION_KEY_AND_VALUE"),
            "Required Annotation",
            ["im-a-key=and a value"]
    )

    // "Required Image Label"

    static final private REQUIRED_IMAGE_LABEL_KEY_ONLY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_IMAGE_LABEL_KEY_ONLY"),
            "Required Image Label",
            ["test.com-i-am-a-key="]
    )

    static final private REQUIRED_IMAGE_LABEL_KEY_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_IMAGE_LABEL_KEY_AND_VALUE"),
            "Required Image Label",
            ["test.com-i-am-another-key=another value"]
    )

    static final private REQUIRED_IMAGE_LABEL_NO_MATCH_I = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_IMAGE_LABEL_NO_MATCH_I"),
            "Required Image Label",
            ["no.match-key="]
    )

    static final private REQUIRED_IMAGE_LABEL_NO_MATCH_II = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_IMAGE_LABEL_NO_MATCH_II"),
            "Required Image Label",
            ["no.match-key=a value"]
    )

    // "Required Label"

    static final private REQUIRED_LABEL_KEY_ONLY = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_LABEL_KEY_ONLY"),
            "Required Label",
            ["im-a-key="]
    )

    static final private REQUIRED_LABEL_KEY_AND_VALUE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_REQUIRED_LABEL_KEY_AND_VALUE"),
            "Required Label",
            ["im-a-key=and_a_value"]
    )

    // "Unscanned Image"

    static final private IMAGES_ARE_UNSCANNED = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_IMAGES_ARE_UNSCANNED"),
            "Unscanned Image",
            ["true"]
    )

    // "Volume Destination"

    static final private NO_FOO_VOLUME_DESTINATIONS = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_FOO_VOLUME_DESTINATIONS"),
            "Volume Destination",
            ["/tmp/foo-volume"]
    )

    // "Volume Name"

    static final private NO_FOO_VOLUME_NAME = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_FOO_VOLUME_NAME"),
            "Volume Name",
            ["foo-volume"]
    )

    // "Volume Source"

    static final private NO_TMP_VOLUME_SOURCE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_TMP_VOLUME_SOURCE"),
            "Volume Source",
            ["/tmp"]
    )

    // "Volume Type"

    static final private NO_HOSTPATH_VOLUME_TYPE = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_HOSTPATH_VOLUME_TYPE"),
            "Volume Type",
            ["HostPath"]
    )

    // "Writable Host Mount"

    static final private NO_READONLY_HOST_MOUNT = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_READONLY_HOST_MOUNT"),
            "Writable Host Mount",
            ["false"]
    )

    // "Writable Mounted Volume"

    static final private NO_WRITABLE_MOUNTED_VOLUMES = setPolicyFieldANDValues(
            BASE_POLICY.clone().setName("AAA_NO_WRITABLE_MOUNTED_VOLUMES"),
            "Writable Mounted Volume",
            ["true"]
    )

    static final private POLICIES = [
            NO_ADD_CAPS_NET_ADMIN,
            NO_ADD_CAPS_SYSLOG,
            NO_ADD_CAPS_NET_ADMIN_AND_SYSLOG,
            NO_ADD_CAPS_LEASE,
            NO_ADD_CAPS_NET_ADMIN_AND_LEASE,
            EXCLUDE_CVE_2019_5436,
            EXCLUDE_CVSS_GT_8,
            CPU_LIMIT_GT_0PT7,
            CPU_LIMIT_GE_1,
            CPU_REQUEST_LT_HALF,
            CPU_REQUEST_GT_HALF,
            MEMORY_LIMIT_LE_750MI,
            MEMORY_LIMIT_GE_750MI,
            MEMORY_REQUEST_EQ_250MI,
            DISALLOWED_ANNOTATION_KEY_ONLY,
            DISALLOWED_ANNOTATION_KEY_AND_VALUE,
            DISALLOWED_IMAGE_LABEL_KEY_ONLY,
            DISALLOWED_IMAGE_LABEL_KEY_AND_VALUE,
            DISALLOWED_IMAGE_LABEL_NO_MATCH_I,
            DISALLOWED_IMAGE_LABEL_NO_MATCH_II,
            HAS_DROP_CAPS_IPC_LOCK,
            HAS_DROP_CAPS_WAKE_ALARM,
            HAS_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM,
            HAS_DROP_CAPS_LEASE,
            HAS_DROP_CAPS_IPC_LOCK_AND_LEASE,
            HAS_RAW_ENV,
            HAS_RAW_ENV_AND_VALUE,
            HAS_ENV_FROM_CONFIG_MAP_KEY,
            HAS_ENV_FROM_SECRET_KEY,
            HAS_ENV_FROM_FIELD,
            HAS_ENV_FROM_RESOURCE_FIELD,
            IS_GREATER_THAN_1_DAY,
            IS_GREATER_THAN_10_YEARS,
            HAS_COMPONENT_CPIO,
            HAS_COMPONENT_CPIO_WITH_VERSION,
            HAS_COMPONENT_CPIO_WITH_OTHER_VERSION,
            IS_BASED_ON_DEBIAN_7,
            IS_BASED_ON_CENTOS_8,
            IS_BASED_ON_ALPINE,
            NO_IMAGE_REGISTRY_USCGR,
            NO_IMAGE_REMOTE,
            NO_IMAGE_TAG,
            NO_OLD_IMAGE_SCANS,
            MINIMUM_RBAC_CLUSTER_WIDE,
            HAS_PORT_25_EXPOSED,
            HAS_EXTERNAL_EXPOSURE,
            IS_PRIVILEGED,
            HAS_BASH_PARENT,
            HAS_VERSION_ARGS,
            HAS_BASH_EXEC,
            HAS_PROCESS_UID_1,
            HAS_RW_ROOT_FS,
            REQUIRED_ANNOTATION_KEY_ONLY,
            REQUIRED_ANNOTATION_KEY_AND_VALUE,
            REQUIRED_IMAGE_LABEL_KEY_ONLY,
            REQUIRED_IMAGE_LABEL_KEY_AND_VALUE,
            REQUIRED_IMAGE_LABEL_NO_MATCH_I,
            REQUIRED_IMAGE_LABEL_NO_MATCH_II,
            REQUIRED_LABEL_KEY_ONLY,
            REQUIRED_LABEL_KEY_AND_VALUE,
            IMAGES_ARE_UNSCANNED,
            NO_FOO_VOLUME_DESTINATIONS,
            NO_FOO_VOLUME_NAME,
            NO_TMP_VOLUME_SOURCE,
            NO_HOSTPATH_VOLUME_TYPE,
            NO_READONLY_HOST_MOUNT,
            NO_WRITABLE_MOUNTED_VOLUMES,
    ]*.build()

    @Shared
    private List<String> createdPolicyIds

    def setupSpec() {
        createdPolicyIds = []
        for (policy in POLICIES) {
            String policyID = CreatePolicyService.createNewPolicy(policy)
            assert policyID
            createdPolicyIds.add(policyID)
        }

        orchestrator.createConfigMap(CONFIG_MAP_NAME, CONFIG_MAP_DATA)
        orchestrator.createSecret(SECRET_NAME)

        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        orchestrator.createDeploymentNoWait(UNSCANNED)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(UNSCANNED)

        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }

        for (policyID in createdPolicyIds) {
            CreatePolicyService.deletePolicy(policyID)
        }
    }

    @SuppressWarnings('LineLength')
    @Unroll
    @Category([BAT])
    def "Expect violation for policy field '#fieldName' - #testName"() {
        // ROX-5298 - Policy tests are unreliable on Openshift
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        expect:
        "Verify expected violations are triggered"
        assert waitForViolation(deployment.name, policy.name, WAIT_FOR_VIOLATION_TIMEOUT)

        where:
        fieldName                   | policy                               | deployment                             | testName
        "Add Capabilities"          | NO_ADD_CAPS_NET_ADMIN                | WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | "match first"
        "Add Capabilities"          | NO_ADD_CAPS_SYSLOG                   | WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | "match last"
        "Add Capabilities"          | NO_ADD_CAPS_NET_ADMIN                | WITH_ADD_CAPS_NET_ADMIN                | "match single"
        "Add Capabilities"          | NO_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | "match set"
        "CVE"                       | EXCLUDE_CVE_2019_5436                | WITH_CVE_2019_5436                     | "match"
        "CVSS"                      | EXCLUDE_CVSS_GT_8                    | WITH_CVSS_GT_8                         | "match"
        "Container CPU Limit"       | CPU_LIMIT_GT_0PT7                    | WITH_CPU_LIMIT_ONE                     | "GT"
        "Container CPU Limit"       | CPU_LIMIT_GE_1                       | WITH_CPU_LIMIT_ONE                     | "GE"
        "Container CPU Request"     | CPU_REQUEST_LT_HALF                  | WITH_CPU_REQUEST_QUARTER               | "LT"
        "Container Memory Limit"    | MEMORY_LIMIT_LE_750MI                | WITH_MEMORY_LIMIT_500MI                | "LE"
        "Container Memory Request"  | MEMORY_REQUEST_EQ_250MI              | WITH_MEMORY_REQUEST_250MI              | "EQ"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_ONLY       | WITH_KEY_ONLY_ANNOTATION               | "key only"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_ONLY       | WITH_KEY_AND_VALUE_ANNOTATION          | "key only matches key and value"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_AND_VALUE  | WITH_KEY_AND_VALUE_ANNOTATION          | "key and value"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_KEY_ONLY      | WITH_IMAGE_LABELS                      | "key only"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_KEY_AND_VALUE | WITH_IMAGE_LABELS                      | "key and value"
        "Drop Capabilities"         | HAS_DROP_CAPS_WAKE_ALARM             | WITH_DROP_CAPS_IPC_LOCK                | "mismatch"
        "Drop Capabilities"         | HAS_DROP_CAPS_LEASE                  | WITH_DROP_CAPS_IPC_LOCK                | "mismatch II"
        "Drop Capabilities"         | HAS_DROP_CAPS_LEASE                  | WITH_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM | "mismatch III"
        "Drop Capabilities"         | HAS_DROP_CAPS_WAKE_ALARM             | WITHOUT_DROP_CAPS                      | "no drops"
        "Drop Capabilities"         | HAS_DROP_CAPS_LEASE                  | WITHOUT_DROP_CAPS                      | "no drops II"
        "Drop Capabilities"         | HAS_DROP_CAPS_IPC_LOCK_AND_LEASE     | WITHOUT_DROP_CAPS                      | "no drops III"
        "Environment Variable"      | HAS_RAW_ENV                          | WITH_RAW_ENV_AND_VALUE                 | "match key"
        "Environment Variable"      | HAS_RAW_ENV                          | WITH_RAW_ENV_AND_DIFFERENT_VALUE       | "match key II"
        "Environment Variable"      | HAS_RAW_ENV_AND_VALUE                | WITH_RAW_ENV_AND_VALUE                 | "match key and value"
        "Environment Variable"      | HAS_ENV_FROM_CONFIG_MAP_KEY          | WITH_ENV_FROM_CONFIG_MAP_KEY           | "match config map key"
        "Environment Variable"      | HAS_ENV_FROM_SECRET_KEY              | WITH_ENV_FROM_SECRET_KEY               | "match secret key"
        "Environment Variable"      | HAS_ENV_FROM_FIELD                   | WITH_ENV_FROM_FIELD                    | "match field"
        "Environment Variable"      | HAS_ENV_FROM_RESOURCE_FIELD          | WITH_ENV_FROM_RESOURCE_FIELD           | "match resource field"
        "Image Age"                 | IS_GREATER_THAN_1_DAY                | OLDER_THAN_1_DAY                       | "match"
        "Image Component"           | HAS_COMPONENT_CPIO                   | WITH_COMPONENT_CPIO                    | "match name"
        "Image Component"           | HAS_COMPONENT_CPIO_WITH_VERSION      | WITH_COMPONENT_CPIO                    | "match name & version"
        "Image OS"                  | IS_BASED_ON_DEBIAN_7                 | BASED_ON_DEBIAN_7                      | "match"
        "Image OS"                  | IS_BASED_ON_CENTOS_8                 | BASED_ON_CENTOS_8                      | "match"
        "Image OS"                  | IS_BASED_ON_ALPINE                   | BASED_ON_ALPINE                        | "match"
        "Image Registry"            | NO_IMAGE_REGISTRY_USCGR              | USES_USGCR                             | "match"
        "Image Remote"              | NO_IMAGE_REMOTE                      | WITH_IMAGE_REMOTE_TO_MATCH             | "match"
        "Image Tag"                 | NO_IMAGE_TAG                         | WITH_IMAGE_TAG_TO_MATCH                | "match"
        //"Image Scan Age"       | NO_OLD_IMAGE_SCANS | UNSCANNED | "match"
        "Minimum RBAC Permissions"  | MINIMUM_RBAC_CLUSTER_WIDE            | SENSOR                                 | "match"
        "Exposed Port"              | HAS_PORT_25_EXPOSED                  | WITH_PORT_25_EXPOSED                   | "match"
        "Port Exposure Method"      | HAS_EXTERNAL_EXPOSURE                | WITH_LB_SERVICE                        | "match"
        "Privileged Container"      | IS_PRIVILEGED                        | WITH_PRIVILEGE                         | "match"
        "Process Ancestor"          | HAS_BASH_PARENT                      | WITH_BASH_PARENT                       | "match"
        "Process Arguments"         | HAS_VERSION_ARGS                     | WITH_VERSION_ARG                       | "match"
        "Process Name"              | HAS_BASH_EXEC                        | WITH_BASH_EXEC                         | "match"
        "Process UID"               | HAS_PROCESS_UID_1                    | WITH_PROCESS_UID_1                     | "match"
        "Read-Only Root Filesystem" | HAS_RW_ROOT_FS                       | WITH_RW_ROOT_FS                        | "match"
        "Required Annotation"       | REQUIRED_ANNOTATION_KEY_AND_VALUE    | WITH_KEY_ONLY_ANNOTATION               | "no key only when value required"
        "Required Annotation"       | REQUIRED_ANNOTATION_KEY_AND_VALUE    | WITH_MISMATCHED_ANNOTATIONS            | "both required"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_KEY_ONLY        | WITHOUT_IMAGE_LABELS                   | "no labels I"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_KEY_AND_VALUE   | WITHOUT_IMAGE_LABELS                   | "no labels II"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_NO_MATCH_I      | WITH_IMAGE_LABELS                      | "no match"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_NO_MATCH_II     | WITH_IMAGE_LABELS                      | "no match II"
        "Required Label"            | REQUIRED_LABEL_KEY_AND_VALUE         | WITH_KEY_ONLY_LABEL                    | "no key only when value required"
        "Required Label"            | REQUIRED_LABEL_KEY_AND_VALUE         | WITH_MISMATCHED_LABELS                 | "both required"
        "Unscanned Image"           | IMAGES_ARE_UNSCANNED                 | UNSCANNED                              | "match"
        "Volume Destination"        | NO_FOO_VOLUME_DESTINATIONS           | WITH_A_RW_FOO_VOLUME                   | "match"
        "Volume Name"               | NO_FOO_VOLUME_NAME                   | WITH_A_RW_FOO_VOLUME                   | "match"
        "Volume Source"             | NO_TMP_VOLUME_SOURCE                 | WITH_A_RO_HOST_BAR_VOLUME              | "match"
        "Volume Type"               | NO_HOSTPATH_VOLUME_TYPE              | WITH_A_RO_HOST_BAR_VOLUME              | "match"
        "Writable Host Mount"       | NO_READONLY_HOST_MOUNT               | WITH_A_RO_HOST_BAR_VOLUME              | "match"
        "Writable Mounted Volume"   | NO_WRITABLE_MOUNTED_VOLUMES          | WITH_A_RW_FOO_VOLUME                   | "match"
    }

    @SuppressWarnings('LineLength')
    @Unroll
    @Category([BAT])
    def "Expect no violation for policy field '#fieldName' - #testName"() {
        // ROX-5298 - Policy tests are unreliable on Openshift
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        expect:
        "Verify unexpected violations are not triggered"
        def violations = AlertService.getViolations(ListAlertsRequest.newBuilder()
                .setQuery("Deployment:${deployment.name}+Policy:${policy.name}").build())
        assert violations.size() == 0

        where:
        fieldName                   | policy                                | deployment                             | testName
        "Add Capabilities"          | NO_ADD_CAPS_LEASE                     | WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | "no match"
        "Add Capabilities"          | NO_ADD_CAPS_LEASE                     | WITH_ADD_CAPS_NET_ADMIN                | "no match II"
        "Add Capabilities"          | NO_ADD_CAPS_NET_ADMIN_AND_LEASE       | WITH_ADD_CAPS_NET_ADMIN_AND_SYSLOG     | "incomplete"
        "Add Capabilities"          | NO_ADD_CAPS_NET_ADMIN_AND_LEASE       | WITH_ADD_CAPS_NET_ADMIN                | "incomplete II"
        "Add Capabilities"          | NO_ADD_CAPS_SYSLOG                    | WITHOUT_ADD_CAPS                       | "missing"
        "CVE"                       | EXCLUDE_CVE_2019_5436                 | WITHOUT_CVE_2019_5436                  | "no match"
        "CVSS"                      | EXCLUDE_CVSS_GT_8                     | WITH_CVSS_LT_8                         | "no match"
        "Container CPU Limit"       | CPU_LIMIT_GT_0PT7                     | WITH_CPU_LIMIT_HALF                    | "not GT"
        "Container CPU Limit"       | CPU_LIMIT_GE_1                        | WITH_CPU_LIMIT_HALF                    | "not GE"
        "Container CPU Limit"       | CPU_LIMIT_GT_0PT7                     | WITHOUT_CPU_LIMIT                      | "missing"
        "Container CPU Request"     | CPU_REQUEST_LT_HALF                   | WITH_CPU_REQUEST_HALF                  | "not LT"
        "Container CPU Request"     | CPU_REQUEST_GT_HALF                   | WITHOUT_CPU_REQUEST                    | "missing"
        "Container Memory Limit"    | MEMORY_LIMIT_LE_750MI                 | WITH_MEMORY_LIMIT_ONEGI                | "not LE"
        "Container Memory Limit"    | MEMORY_LIMIT_GE_750MI                 | WITHOUT_MEMORY_LIMIT                   | "missing"
        "Container Memory Request"  | MEMORY_REQUEST_EQ_250MI               | WITH_MEMORY_REQUEST_HALFGI             | "not EQ"
        "Container Memory Request"  | MEMORY_REQUEST_EQ_250MI               | WITHOUT_MEMORY_REQUEST                 | "missing"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_AND_VALUE   | WITH_KEY_ONLY_ANNOTATION               | "no key only when value"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_AND_VALUE   | WITH_MISMATCHED_ANNOTATIONS            | "both required"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_ONLY        | WITHOUT_ANNOTATIONS                    | "missing"
        "Disallowed Annotation"     | DISALLOWED_ANNOTATION_KEY_AND_VALUE   | WITHOUT_ANNOTATIONS                    | "missing"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_KEY_ONLY       | WITHOUT_IMAGE_LABELS                   | "no labels I"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_KEY_AND_VALUE  | WITHOUT_IMAGE_LABELS                   | "no labels II"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_NO_MATCH_I     | WITH_IMAGE_LABELS                      | "no match"
        "Disallowed Image Label"    | DISALLOWED_IMAGE_LABEL_NO_MATCH_II    | WITH_IMAGE_LABELS                      | "no match II"
        "Drop Capabilities"         | HAS_DROP_CAPS_IPC_LOCK                | WITH_DROP_CAPS_IPC_LOCK                | "has drop"
        "Drop Capabilities"         | HAS_DROP_CAPS_IPC_LOCK                | WITH_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM | "has drop II"
        "Drop Capabilities"         | HAS_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM | WITH_DROP_CAPS_IPC_LOCK_AND_WAKE_ALARM | "has drop III"
        "Environment Variable"      | HAS_RAW_ENV_AND_VALUE                 | WITH_RAW_ENV_AND_DIFFERENT_VALUE       | "has key but different value"
        "Environment Variable"      | HAS_RAW_ENV_AND_VALUE                 | WITHOUT_ENV                            | "no env to match"
        "Environment Variable"      | HAS_RAW_ENV                           | WITHOUT_ENV                            | "no env to match II"
        "Environment Variable"      | HAS_ENV_FROM_CONFIG_MAP_KEY           | WITH_DIFFERENT_ENV_FROM_CONFIG_MAP_KEY | "no match config map key"
        "Environment Variable"      | HAS_ENV_FROM_CONFIG_MAP_KEY           | WITHOUT_ENV                            | "no env to match III"
        "Environment Variable"      | HAS_ENV_FROM_SECRET_KEY               | WITH_DIFFERENT_ENV_FROM_SECRET_KEY     | "no match secret key"
        "Environment Variable"      | HAS_ENV_FROM_SECRET_KEY               | WITHOUT_ENV                            | "no env to match IV"
        "Environment Variable"      | HAS_ENV_FROM_FIELD                    | WITH_DIFFERENT_ENV_FROM_FIELD          | "no match field"
        "Environment Variable"      | HAS_ENV_FROM_FIELD                    | WITHOUT_ENV                            | "no env to match V"
        "Environment Variable"      | HAS_ENV_FROM_RESOURCE_FIELD           | WITH_DIFFERENT_ENV_FROM_RESOURCE_FIELD | "no match resource field"
        "Environment Variable"      | HAS_ENV_FROM_RESOURCE_FIELD           | WITHOUT_ENV                            | "no env to match VI"
        "Image Age"                 | IS_GREATER_THAN_10_YEARS              | YOUNGER_THAN_TEN_YEARS                 | "no match"
        "Image Component"           | HAS_COMPONENT_CPIO                    | WITHOUT_COMPONENT_CPIO                 | "no match"
        "Image Component"           | HAS_COMPONENT_CPIO_WITH_OTHER_VERSION | WITH_COMPONENT_CPIO                    | "no match on version"
        "Image OS"                  | IS_BASED_ON_ALPINE                    | BASED_ON_CENTOS_8                      | "no match"
        "Image Registry"            | NO_IMAGE_REGISTRY_USCGR               | USES_DOCKER                            | "no match"
        "Image Remote"              | NO_IMAGE_REMOTE                       | WITH_IMAGE_REMOTE_TO_NOT_MATCH         | "no match"
        "Image Tag"                 | NO_IMAGE_TAG                          | WITH_IMAGE_TAG_TO_NOT_MATCH            | "no match"
        "Image Scan Age"            | NO_OLD_IMAGE_SCANS                    | WITH_RECENT_SCAN_AGE                   | "no match"
        "Minimum RBAC Permissions"  | MINIMUM_RBAC_CLUSTER_WIDE             | CENTRAL                                | "no match"
        "Exposed Port"              | HAS_PORT_25_EXPOSED                   | WITHOUT_PORTS_EXPOSED                  | "no match"
        "Port Exposure Method"      | HAS_EXTERNAL_EXPOSURE                 | WITHOUT_SERVICE                        | "no match"
        "Privileged Container"      | IS_PRIVILEGED                         | WITHOUT_PRIVILEGE                      | "no match"
        "Process Ancestor"          | HAS_BASH_PARENT                       | WITHOUT_BASH_PARENT                    | "no match"
        "Process Arguments"         | HAS_VERSION_ARGS                      | WITHOUT_VERSION_ARG                    | "no match"
        "Process Name"              | HAS_BASH_EXEC                         | WITHOUT_BASH_EXEC                      | "no match"
        "Process UID"               | HAS_PROCESS_UID_1                     | WITHOUT_PROCESS_UID_1                  | "no match"
        "Read-Only Root Filesystem" | HAS_RW_ROOT_FS                        | WITH_RDONLY_ROOT_FS                    | "no match"
        "Required Annotation"       | REQUIRED_ANNOTATION_KEY_ONLY          | WITH_KEY_ONLY_ANNOTATION               | "key only"
        "Required Annotation"       | REQUIRED_ANNOTATION_KEY_ONLY          | WITH_KEY_AND_VALUE_ANNOTATION          | "key only matches key and value"
        "Required Annotation"       | REQUIRED_ANNOTATION_KEY_AND_VALUE     | WITH_KEY_AND_VALUE_ANNOTATION          | "key and value"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_KEY_ONLY         | WITH_IMAGE_LABELS                      | "key only"
        "Required Image Label"      | REQUIRED_IMAGE_LABEL_KEY_AND_VALUE    | WITH_IMAGE_LABELS                      | "key and value"
        "Required Label"            | REQUIRED_LABEL_KEY_ONLY               | WITH_KEY_ONLY_LABEL                    | "key only"
        "Required Label"            | REQUIRED_LABEL_KEY_ONLY               | WITH_KEY_AND_VALUE_LABEL               | "key only matches key and value"
        "Required Label"            | REQUIRED_LABEL_KEY_AND_VALUE          | WITH_KEY_AND_VALUE_LABEL               | "key and value"
        "Unscanned Image"           | IMAGES_ARE_UNSCANNED                  | IS_SCANNED                             | "no match"
        "Volume Destination"        | NO_FOO_VOLUME_DESTINATIONS            | WITH_A_RO_HOST_BAR_VOLUME              | "no match"
        "Volume Destination"        | NO_FOO_VOLUME_DESTINATIONS            | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
        "Volume Name"               | NO_FOO_VOLUME_NAME                    | WITH_A_RO_HOST_BAR_VOLUME              | "no match"
        "Volume Name"               | NO_FOO_VOLUME_NAME                    | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
        "Volume Source"             | NO_TMP_VOLUME_SOURCE                  | WITH_A_RW_FOO_VOLUME                   | "no match"
        "Volume Source"             | NO_TMP_VOLUME_SOURCE                  | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
        "Volume Type"               | NO_HOSTPATH_VOLUME_TYPE               | WITH_A_RW_FOO_VOLUME                   | "no match"
        "Volume Type"               | NO_HOSTPATH_VOLUME_TYPE               | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
        "Writable Host Mount"       | NO_READONLY_HOST_MOUNT                | WITH_A_RW_FOO_VOLUME                   | "no match"
        "Writable Host Mount"       | NO_READONLY_HOST_MOUNT                | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
        "Writable Mounted Volume"   | NO_WRITABLE_MOUNTED_VOLUMES           | WITH_A_RO_HOST_BAR_VOLUME              | "no match"
        "Writable Mounted Volume"   | NO_WRITABLE_MOUNTED_VOLUMES           | WITHOUT_FOO_OR_BAR_VOLUMES             | "no match II"
    }

    private static setPolicyFieldANDValues(Policy.Builder builder, String fieldName, List<String> values) {
        def policyGroup = PolicyGroup.newBuilder()
                .setFieldName(fieldName)
                .setBooleanOperator(PolicyOuterClass.BooleanOperator.AND)
        policyGroup.addAllValues(values.collect { PolicyValue.newBuilder().setValue(it).build() }).build()
        return builder.clone().addPolicySections(
                PolicySection.newBuilder().addPolicyGroups(policyGroup.build()).build()
        )
    }
}

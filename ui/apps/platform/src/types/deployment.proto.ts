import { ContainerRuntimeType } from './containerRuntime.proto';
import { ImageName } from './image.proto';
import { MatchLabelsSelector } from './labels.proto';
import { PermissionLevel } from './rbac.proto';
import { Toleration } from './taints.proto';

export type ListDeployment = {
    id: string;
    hash: string; // uint64
    name: string;
    cluster: string;
    clusterId: string;
    namespace: string;
    created: string; // ISO 8601 date string
    priority: string; // int64
};

export type Deployment = {
    id: string;
    name: string;
    hash: string; // uint64
    type: string;
    namespace: string;
    namespaceId: string;
    orchestratorComponent: boolean;
    replicas: string; // int64
    labels: Record<string, string>;
    podLabels: Record<string, string>;
    labelSelector: MatchLabelsSelector;
    created: string; // ISO 8601 date string
    clusterId: string;
    clusterName: string;
    containers: Container[];
    annotations: Record<string, string>;
    priority: string; // int64
    inactive: boolean;
    imagePullSecrets: string[];
    serviceAccount: string;
    serviceAccountPermissionLevel: PermissionLevel;
    automountServiceAccountToken: boolean;
    hostNetwork: boolean;
    hostPid: boolean;
    hostIpc: boolean;
    tolerations: Toleration[];
    ports: PortConfig[];
    stateTimestamp: string; // int64
    riskScore: number; // float
};

export type Container = {
    id: string;
    config: ContainerConfig;
    image: ContainerImage;
    securityContext: ContainerSecurityContext;
    volumes: ContainerVolume[];
    ports: PortConfig[];
    secrets: EmbeddedSecret[];
    resources: ContainerResources;
    name: string;
};

export type ContainerConfig = {
    env: EnvironmentConfig[];
    command: string[];
    args: string[];
    directory: string;
    user: string;
    uid: string; // int64
    appArmorProfile: string;
};

export type EnvironmentConfig = {
    key: string;
    value: string;
    envVarSource: EnvVarSource;
};

export type EnvVarSource =
    | 'UNSET'
    | 'RAW'
    | 'SECRET_KEY'
    | 'CONFIG_MAP_KEY'
    | 'FIELD'
    | 'RESOURCE_FIELD'
    | 'UNKNOWN';

export type ContainerImage = {
    id: string;
    name: ImageName;
    notPullable: boolean;
};

export type ContainerSecurityContext = {
    privileged: boolean;
    selinux: SELinux | null;
    dropCapabilities: string[];
    addCapabilities: string[];
    readOnlyRootFilesystem: boolean;
    seccompProfile: SeccompProfile | null;
};

export type SELinux = {
    user: string;
    role: string;
    type: string;
    level: string;
};

export type SeccompProfile = {
    type: SeccompProfileType;
    localhostProfile: string;
};

export type SeccompProfileType = 'UNCONFINED' | 'RUNTIME_DEFAULT' | 'LOCALHOST';

export type ContainerVolume = {
    name: string;
    source: string;
    destination: string;
    readOnly: boolean;
    type: string;
    mountPropagation: MountPropagation;
};

export type MountPropagation = 'NONE' | 'HOST_TO_CONTAINER' | 'BIDIRECTIONAL';

export type PortConfig = {
    name: string;
    containerPort: number; // int32
    protocol: string;
    exposure: ExposureLevel;
    exposedPort: number; // int32 deprecated
    exposureInfos: ExposureInfo[];
};

export type ExposureLevel = 'UNSET' | 'EXTERNAL' | 'NODE' | 'INTERNAL' | 'HOST';

export type ExposureInfo = {
    level: ExposureLevel;

    // only set if level is not HOST
    serviceName: string;
    serviceId: string;
    serviceClusterIp: string;
    servicePort: number; // int32

    // only set if level is HOST, NODE, or EXTERNAL
    nodePort: number; // int32

    // only set if level is EXTERNAL
    externalIps: string[];
    externalHostnames: string[];
};

export type EmbeddedSecret = {
    name: string;
    path: string;
};

export type ContainerResources = {
    cpuCoresRequest: number;
    cpuCoresLimit: number;
    memoryMbRequest: number;
    memoryMbLimit: number;
};

// Pod represents information for a currently running pod or deleted pod in an active deployment.
export type Pod = {
    id: string;
    name: string;
    deploymentId: string;
    namespace: string;
    clusterId: string;
    liveInstances: ContainerInstance[];
    // Must be a list of lists, so we can perform search queries (does not work for maps that aren't <string, string>)
    // There is one bucket (list) per container name.
    terminatedInstances: ContainerInstanceList[];
    started: string; // ISO 8601 date string Time Kubernetes reports the pod was created.
};

export type ContainerInstance = {
    instanceId: ContainerInstanceID;
    containingPodId: string; // The pod containing this container instance (kubernetes only).
    containerName: string;
    containerIps: string[];
    started: string; // ISO 8601 date string
    imageDigest: string; // Image ID
    finished: string; // ISO 8601 date string The finish time of the container, if it finished.
    exitCode: number; // int32 The exit code of the container. Only valid when finished is populated.
    terminationReason: string; // The reason for the container's termination, if it finished.
};

// ContainerInstanceID allows to uniquely identify a container within a cluster.
export type ContainerInstanceID = {
    containerRuntime: ContainerRuntimeType;
    id: string;
    node: string;
};

export type ContainerInstanceList = {
    instances: ContainerInstance[];
};

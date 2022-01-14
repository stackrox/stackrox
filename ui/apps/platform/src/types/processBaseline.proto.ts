export type ProcessBaselineKey = {
    deploymentId: string;
    containerName: string;
    clusterId: string;
    namespace: string;
};

export type ProcessBaseline = {
    id: string;
    key: ProcessBaselineKey;

    elements: ProcessBaselineElement[];
    elementGraveyard: ProcessBaselineElement[];

    created: string; // ISO 8601 date string
    userLockedTimestamp: string; // ISO 8601 date string
    stackRoxLockedTimestamp: string; // ISO 8601 date string
    lastUpdate: string; // ISO 8601 date string
};

export type ProcessBaselineElement = {
    element: ProcessBaselineItem;
    auto: boolean;
};

export type ProcessBaselineItem = {
    processName?: string;
};

export type ContainerNameAndBaselineStatus = {
    containerName: string;
    baselineStatus: ProcessBaselineStatus;
    anomalousProcessesExecuted: boolean;
};

export type ProcessBaselineStatus = 'INVALID' | 'NOT_GENERATED' | 'UNLOCKED' | 'LOCKED';

export type ProcessBaselineResults = {
    deploymentId: string;
    clusterId: string;
    namespace: string;
    baselineStatuses: ContainerNameAndBaselineStatus[];
};

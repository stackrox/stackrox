export type SensorHealthStatus = 'HEALTHY' | 'UNHEALTHY' | 'DEGRADED' | 'UNINITIALIZED';

export type ClusterHealthItemStatus =
    | 'HEALTHY'
    | 'UNHEALTHY'
    | 'DEGRADED'
    | 'UNINITIALIZED'
    | 'UNAVAILABLE';

export type ClusterHealthStatus = {
    admissionControlHealthStatus?: ClusterHealthItemStatus;
    admissionControlHealthInfo?: {
        totalDesiredPods: number;
        totalReadyPods: number;
        statusErrors: string[];
    };
    collectorHealthStatus?: ClusterHealthItemStatus;
    collectorHealthInfo?: {
        version: string;
        totalDesiredPods: number;
        totalReadyPods: number;
        totalRegisteredNodes: number;
        statusErrors: string[];
    };
    sensorHealthStatus: SensorHealthStatus;
    overallHealthStatus: SensorHealthStatus;
    healthInfoComplete: boolean;
    lastContact: string; // ISO 8601
};

export type ClusterHealthItem = 'collector' | 'sensor' | 'admissionControl' | 'scanner';

export type SensorUpgradeStatus = {
    upgradability: string;
    upgradabilityStatusReason: string;
    mostRecentProcess: {
        active: boolean;
        progress: {
            upgradeState: string;
            upgradeStatusDetail: string;
        };
        type: string;
    };
};

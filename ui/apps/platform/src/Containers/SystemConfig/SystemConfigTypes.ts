type Config = {
    backgroundColor: string;
    color: string;
    enabled: boolean;
    size: string;
    text: 'UNSET' | 'SMALL' | 'MEDIUM' | 'LARGE';
};

export type PrivateConfig = {
    alertConfig: {
        allRuntimeRetentionDurationDays: number;
        attemptedDeployRetentionDurationDays: number;
        attemptedRuntimeRetentionDurationDays: number;
        deletedRuntimeRetentionDurationDays: number;
        resolvedDeployRetentionDurationDays: number;
    };
    imageRetentionDurationDays: number;
    expiredVulnReqRetentionDurationDays: number;
};

export type PublicConfig = {
    footer: Config;
    header: Config;
    loginNotice?: {
        enabled: boolean;
        text: string;
    };
};

export type TelemetryConfig = {
    enabled: boolean;
    lastSetTime: string;
};

export type SystemConfig = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
};

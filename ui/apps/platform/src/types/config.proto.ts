export type Telemetry = {
    enabled: boolean;
};

export type LoginNotice = {
    enabled: boolean;
    text: string;
};

export type BannerConfigSize = 'UNSET' | 'SMALL' | 'MEDIUM' | 'LARGE';

export type BannerConfig = {
    enabled: boolean;
    text: string;
    size: BannerConfigSize;
    color: string;
    backgroundColor: string;
};

export type PublicConfig = {
    /*
     * GET /v1/config/public response can have any of the following null if unitialized.
     */
    loginNotice: LoginNotice | null;
    header: BannerConfig | null;
    footer: BannerConfig | null;
    telemetry: Telemetry | null;
};

export type AlertRetentionConfig = {
    resolvedDeployRetentionDurationDays: number; // int32
    deletedRuntimeRetentionDurationDays: number; // int32
    allRuntimeRetentionDurationDays: number; // int32
    attemptedDeployRetentionDurationDays: number; // int32
    attemptedRuntimeRetentionDurationDays: number; // int32
};

export type DecommissionedClusterRetentionConfig = {
    retentionDurationDays: number; // int32
    ignoreClusterLabels: Record<string, string>;
    lastUpdated: string; // ISO 8601 date string
    createdAt: string; // ISO 8601 date string
};

export type ReportRetentionConfig = {
    historyRetentionDurationDays: number; // uint32
    downloadableReportGlobalRetentionBytes: number; // uint32
    downloadableReportRetentionDays: number; // uint32
};

export type AdministrationEventsConfig = {
    retentionDurationDays: number; // unit32
};

export type PrivateConfig = {
    alertConfig: AlertRetentionConfig;
    imageRetentionDurationDays: number; // int32
    expiredVulnReqRetentionDurationDays: number; // int32
    decommissionedClusterRetention: DecommissionedClusterRetentionConfig;
    reportRetentionConfig: ReportRetentionConfig;
    administrationEventsConfig: AdministrationEventsConfig;
};

export type SystemConfig = {
    /*
     * GET /v1/config response can have publicConfig: null if uninitialized.
     */
    publicConfig: PublicConfig | null;
    privateConfig: PrivateConfig;
};

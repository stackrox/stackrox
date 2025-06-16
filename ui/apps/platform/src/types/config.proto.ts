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
    retentionDurationDays: number; // uint32
};

export type Condition = {
    operator: string;
    argument: string;
};

export type Expression = {
    expression?: Condition[];
};

export enum Exposure {
          NONE = 0;
      INTERNAL = 1;
      EXTERNAL = 2;
      BOTH = 3;

}

export type Labels = {
    labels: Record<string, Expression>;
    exposure: Exposure;
    registryName: string;
};

export type Metrics = {
    gatheringPeriodMinutes?: number; // uint32
    metrics?: Record<string, Labels>;
    filter?: string;
};

// The type list of known metrics categories.
export type Category = 'imageVulnerabilities';

export type PrometheusMetricsConfig = {
    imageVulnerabilities?: Metrics;
};

export type PrivateConfig = {
    alertConfig: AlertRetentionConfig;
    imageRetentionDurationDays: number; // int32
    expiredVulnReqRetentionDurationDays: number; // int32
    decommissionedClusterRetention: DecommissionedClusterRetentionConfig;
    reportRetentionConfig: ReportRetentionConfig;
    administrationEventsConfig: AdministrationEventsConfig;
    prometheusMetricsConfig: PrometheusMetricsConfig;
};

export type PlatformComponentRule = {
    name: string;
    namespaceRule: {
        regex: string;
    };
};

export type PlatformComponentsConfig = {
    needsReevaluation: boolean;
    rules: PlatformComponentRule[];
};

export type SystemConfig = {
    /*
     * GET /v1/config response can have publicConfig: null if uninitialized.
     */
    publicConfig: PublicConfig | null;
    privateConfig: PrivateConfig;
    platformComponentConfig: PlatformComponentsConfig;
};

export type DelegatedRegistryConfigEnabledFor = 'NONE' | 'ALL' | 'SPECIFIC';

export type EnabledSelections = Exclude<DelegatedRegistryConfigEnabledFor, 'NONE'>;

export type DelegatedRegistry = {
    path: string;
    clusterId: string;
};

export type DelegatedRegistryConfig = {
    enabledFor: DelegatedRegistryConfigEnabledFor;
    defaultClusterId: string;
    registries: DelegatedRegistry[];
};

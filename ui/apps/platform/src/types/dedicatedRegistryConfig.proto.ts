export type DelegatedRegistryConfigEnabledFor = 'NONE' | 'ALL' | 'SPECIFIC';

export type DelegatedRegistry = {
    path: string;
    clusterId: string;
};

export type DelegatedRegistryConfig = {
    enabledFor: DelegatedRegistryConfigEnabledFor;
    defaultClusterId: string;
    registries: DelegatedRegistry[];
};

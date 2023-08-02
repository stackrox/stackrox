import axios from './instance';
import { Empty } from './types';

export type DelegatedRegistryConfigEnabledFor = 'NONE' | 'ALL' | 'SPECIFIC';

export type EnabledSelections = Exclude<DelegatedRegistryConfigEnabledFor, 'NONE'>;

export type DelegatedRegistry = {
    path: string;
    clusterId: string;
};

export type DelegatedRegistryCluster = {
    id: string;
    name: string;
    isValid: boolean;
};

export type DelegatedRegistryConfig = {
    enabledFor: DelegatedRegistryConfigEnabledFor;
    defaultClusterId: string;
    registries: DelegatedRegistry[];
};

const delegatedRegistryUrl = '/v1/delegatedregistryconfig';

/**
 * Fetches the declarative config health objects.
 */
export function fetchDelegatedRegistryConfig(): Promise<DelegatedRegistryConfig> {
    return axios
        .get<DelegatedRegistryConfig>(delegatedRegistryUrl)
        .then((response) => response.data);
}

/**
 * Fetches clusters that have local scanning enabled.
 */
export function fetchDelegatedRegistryClusters(): Promise<DelegatedRegistryCluster[]> {
    return axios
        .get<{ clusters: DelegatedRegistryCluster[] }>(`${delegatedRegistryUrl}/clusters`)
        .then((response) => response.data.clusters || []);
}

/**
 * Updates the declarative config health objects.
 */
export function updateDelegatedRegistryConfig(delegatedRegistryConfig): Promise<Empty> {
    return axios.put(delegatedRegistryUrl, delegatedRegistryConfig);
}

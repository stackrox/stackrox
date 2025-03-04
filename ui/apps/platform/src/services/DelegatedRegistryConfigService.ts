import axios from './instance';

// Determines if delegation is enabled for no registries, all registries, or specific registries:
// Scan all images via central services except for images from the OCP integrated registry.
// Scan all images via the secured clusters.
// Scan images that match registries or are from the OCP integrated registry via the secured clusters otherwise scan via central services.
export type DelegatedRegistryConfigEnabledFor = 'NONE' | 'ALL' | 'SPECIFIC';

// Note:
// In order to make each row of registry/cluster exceptions work
// with PatternFly's drag-and-drag Table variant
// we need to add stable surrogate UUIDs to each entry
//
// see description in https://github.com/stackrox/stackrox/pull/7341
// for more details
export type DelegatedRegistry = {
    uuid?: string; // not in API response
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

    // Relationship between enabledFor and registries.
    //
    // NONE: registries has no effect.
    //
    // ALL: registries directs ad-hoc requests to the specified secured clusters if the path matches.
    //
    // SPECIFIC: registries directs ad-hoc requests to the specified secured clusters just like with ALL,
    // but in addition images that match the specified paths will be scanned locally by the secured clusters
    // (images from the OCP integrated registry are always scanned locally).
    // Images that do not match a path will be scanned via central services.
    //
    // ad-hoc requests: roxctl CLI, Jenkins plugin, or API.
    registries: DelegatedRegistry[];
};

const delegatedRegistryUrl = '/v1/delegatedregistryconfig';

/**
 * Fetches the delegated registry configuration.
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
 * Updates the delegated registry configuration.
 */
export function updateDelegatedRegistryConfig(
    delegatedRegistryConfig: DelegatedRegistryConfig
): Promise<DelegatedRegistryConfig> {
    return axios
        .put<DelegatedRegistryConfig>(delegatedRegistryUrl, delegatedRegistryConfig)
        .then((response) => response.data);
}

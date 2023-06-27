import axios from './instance';

import { DelegatedRegistryConfig } from '../types/dedicatedRegistryConfig.proto';

const delegatedRegistryUrl = '/v1/delegatedregistryconfig';

/**
 * Fetches the declarative config health objects.
 */
export function fetchDelegatedRegistryConfig(): Promise<DelegatedRegistryConfig> {
    return axios
        .get<DelegatedRegistryConfig>(delegatedRegistryUrl)
        .then((response) => response.data);
}

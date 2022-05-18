import { PublicConfig, SystemConfig } from '../types/config.proto';

import axios from './instance';

const baseUrl = '/v1/config';

/**
 * Fetches system configurations.
 */
export function fetchSystemConfig(): Promise<{ response: SystemConfig }> {
    return axios.get<SystemConfig>(baseUrl).then(({ data }) => ({
        response: data,
    }));
}

/**
 * Fetches login notice and header/footer info.
 */
export function fetchPublicConfig(): Promise<{ response: PublicConfig }> {
    return axios.get<PublicConfig>(`${baseUrl}/public`).then(({ data }) => ({
        response: data,
    }));
}

/**
 * Saves modified system config.
 */
export function saveSystemConfig(config: SystemConfig): Promise<SystemConfig> {
    return axios.put<SystemConfig>(baseUrl, config).then((response) => response.data);
}

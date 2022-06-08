import { PublicConfig, SystemConfig } from '../types/config.proto';

import axios from './instance';

const baseUrl = '/v1/config';

/*
 * Fetch system configuration: private and public.
 */
export function fetchSystemConfig(): Promise<SystemConfig> {
    return axios.get<SystemConfig>(baseUrl).then((response) => response.data);
}

/*
 * Fetch login notice and header/footer info.
 */
export function fetchPublicConfig(): Promise<{ response: PublicConfig }> {
    return axios.get<PublicConfig>(`${baseUrl}/public`).then(({ data }) => ({
        response: data,
    }));
}

/*
 * Save system configuration: private and public.
 */
export function saveSystemConfig(config: SystemConfig): Promise<SystemConfig> {
    return axios.put<SystemConfig>(baseUrl, { config }).then((response) => response.data);
}

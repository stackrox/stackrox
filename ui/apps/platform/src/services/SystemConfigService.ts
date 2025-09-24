import type { PublicConfig, SystemConfig } from 'types/config.proto';

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
export function fetchPublicConfig(): Promise<PublicConfig> {
    return axios.get<PublicConfig>(`${baseUrl}/public`).then(({ data }) => data);
}

/*
 * Save system configuration: private and public.
 */
export function saveSystemConfig(config: SystemConfig): Promise<SystemConfig> {
    return axios.put<SystemConfig>(baseUrl, { config }).then((response) => response.data);
}

/**
 *  Fetch the default Red Hat layered products namespace regex rule
 */
export function fetchDefaultRedHatLayeredProductsRule(): Promise<string> {
    return axios
        .get<{ regex: string }>(`${baseUrl}/platformcomponent/rhlp/default`)
        .then((response) => {
            return response.data.regex;
        });
}

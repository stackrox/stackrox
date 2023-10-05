import axios from './instance';

const vulnBaseUrl = '/v1/config/private/deferral/vulnerabilities';

export type VulnerabilitiesExceptionConfig = {
    expiryOptions: {
        dayOptions: {
            numDays: number;
            enabled: boolean;
        }[];
        fixableCveOptions: {
            allFixable: boolean;
            anyFixable: boolean;
        };
        customDate: boolean;
    };
};

export function fetchVulnerabilitiesExceptionConfig(): Promise<VulnerabilitiesExceptionConfig | null> {
    return axios
        .get<{ config: VulnerabilitiesExceptionConfig | null }>(vulnBaseUrl)
        .then(({ data }) => data.config);
}

export function updateVulnerabilitiesExceptionConfig(
    config: Partial<VulnerabilitiesExceptionConfig>
): Promise<Partial<VulnerabilitiesExceptionConfig>> {
    return axios
        .put<{ config: Partial<VulnerabilitiesExceptionConfig> }>(`${vulnBaseUrl}`, { config })
        .then(({ data }) => data.config);
}

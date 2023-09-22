import axios from './instance';

const vulnBaseUrl = '/v1/config/private/exception/vulnerabilities';

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
        indefinite: boolean;
    };
};

export function fetchVulnerabilitiesExceptionConfig(): Promise<VulnerabilitiesExceptionConfig> {
    return axios
        .get<{ config: VulnerabilitiesExceptionConfig }>(vulnBaseUrl)
        .then(({ data }) => data.config);
}

export function updateVulnerabilitiesExceptionConfig(
    config: Partial<VulnerabilitiesExceptionConfig>
): Promise<Partial<VulnerabilitiesExceptionConfig>> {
    return axios
        .put<{ config: Partial<VulnerabilitiesExceptionConfig> }>(`${vulnBaseUrl}`, { config })
        .then(({ data }) => data.config);
}

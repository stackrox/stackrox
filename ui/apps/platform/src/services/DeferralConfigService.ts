import axios from './instance';

const vulnBaseUrl = '/v1/config/private/deferral/vulnerabilities';

export type VulnerabilitiesDeferralConfig = {
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

export function fetchVulnerabilitiesDeferralConfig(): Promise<VulnerabilitiesDeferralConfig | null> {
    return axios
        .get<{ config: VulnerabilitiesDeferralConfig | null }>(vulnBaseUrl)
        .then(({ data }) => data.config);
}

export function updateVulnerabilitiesDeferralConfig(
    config: Partial<VulnerabilitiesDeferralConfig>
): Promise<Partial<VulnerabilitiesDeferralConfig>> {
    return axios
        .put<{ config: Partial<VulnerabilitiesDeferralConfig> }>(`${vulnBaseUrl}`, { config })
        .then(({ data }) => data.config);
}

import axios from 'services/instance';

import { complianceV2Url } from './ComplianceCommon';

const complianceIntegrationsBaseUrl = `${complianceV2Url}/integrations`;

export type ComplianceIntegration = {
    id: string;
    version: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    statusErrors: string[];
    operatorInstalled: boolean;
};

/**
 * Fetches a list of clusters available for a compliance scan.
 */
export function listComplianceIntegrations(): Promise<ComplianceIntegration[]> {
    return axios
        .get<{ integrations: ComplianceIntegration[] }>(complianceIntegrationsBaseUrl)
        .then((response) => {
            return response?.data?.integrations ?? [];
        });
}

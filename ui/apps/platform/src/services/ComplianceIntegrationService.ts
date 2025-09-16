import axios from 'services/instance';

import { complianceV2Url } from './ComplianceCommon';

const complianceIntegrationsBaseUrl = `${complianceV2Url}/integrations`;

type COStatus = 'HEALTHY' | 'UNHEALTHY';

type ClusterProviderType = 'UNSPECIFIED' | 'AKS' | 'ARO' | 'EKS' | 'GKE' | 'OCP' | 'OSD' | 'ROSA';

type ClusterPlatformType =
    | 'GENERIC_CLUSTER'
    | 'KUBERNETES_CLUSTER'
    | 'OPENSHIFT_CLUSTER'
    | 'OPENSHIFT4_CLUSTER';

export type ComplianceIntegration = {
    id: string;
    version: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    statusErrors: string[];
    operatorInstalled: boolean;
    status: COStatus;
    clusterPlatformType: ClusterPlatformType;
    clusterProviderType: ClusterProviderType;
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

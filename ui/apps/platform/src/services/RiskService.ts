import type { Deployment } from 'types/deployment.proto';
import axios from './instance';

const riskUrl = '/v1/risk';

export type RiskAdjustmentResponse = {
    deployment: Deployment;
    original_score: number;
    effective_score: number;
    message: string;
};

export type ResetAllRisksResponse = {
    count: number;
    message: string;
};

export function changeDeploymentRiskPosition(
    deploymentId: string,
    aboveDeploymentId?: string,
    belowDeploymentId?: string
): Promise<RiskAdjustmentResponse> {
    return axios
        .post<RiskAdjustmentResponse>(`${riskUrl}/deployment/${deploymentId}/position`, {
            deployment_id: deploymentId,
            above_deployment_id: aboveDeploymentId || '',
            below_deployment_id: belowDeploymentId || '',
        })
        .then((response) => response.data);
}

export function resetDeploymentRisk(deploymentId: string): Promise<RiskAdjustmentResponse> {
    return axios
        .post<RiskAdjustmentResponse>(`${riskUrl}/deployment/${deploymentId}/reset`, {
            deployment_id: deploymentId,
        })
        .then((response) => response.data);
}

export function resetAllDeploymentRisks(): Promise<ResetAllRisksResponse> {
    return axios
        .post<ResetAllRisksResponse>(`${riskUrl}/deployments/reset-all`, {})
        .then((response) => response.data);
}

import type { Risk } from 'types/risk.proto';
import axios from './instance';

const riskUrl = '/v1/risk';

export type RiskPositionDirection = 'RISK_POSITION_UP' | 'RISK_POSITION_DOWN';

export type RiskAdjustmentResponse = {
    risk: Risk;
    message: string;
};

export function changeDeploymentRiskPosition(
    deploymentId: string,
    direction: RiskPositionDirection
): Promise<RiskAdjustmentResponse> {
    return axios
        .post<RiskAdjustmentResponse>(`${riskUrl}/deployment/${deploymentId}/position`, {
            deployment_id: deploymentId,
            direction,
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

import axios from './instance';
import type { Empty } from './types';

const aiIntegrationsUrl = '/v2/ai-integrations';

export type AiIntegrationType = 'AI_INTEGRATION_TYPE_UNSPECIFIED' | 'AI_INTEGRATION_TYPE_OLS';

export type AiIntegration = {
    id: string;
    name: string;
    type: AiIntegrationType;
    serviceUrl: string;
};

// The v2 AI integrations API wraps the resource in request/response messages,
// consistent with the cloud sources service (ListCloudSourcesResponse,
// GetCloudSourceResponse, CreateCloudSourceRequest, ...).
type ListAiIntegrationsResponse = {
    integrations: AiIntegration[];
};

type AiIntegrationResponse = {
    integration: AiIntegration;
};

type AiIntegrationRequest = {
    integration: AiIntegration;
};

export function fetchAiIntegrations(): Promise<AiIntegration[]> {
    return axios
        .get<ListAiIntegrationsResponse>(aiIntegrationsUrl)
        .then((response) => response.data.integrations);
}

export function fetchAiIntegration(id: string): Promise<AiIntegration> {
    return axios
        .get<AiIntegrationResponse>(`${aiIntegrationsUrl}/${id}`)
        .then((response) => response.data.integration);
}

export function createAiIntegration(data: AiIntegration): Promise<AiIntegration> {
    const request: AiIntegrationRequest = { integration: data };
    return axios
        .post<AiIntegrationResponse>(aiIntegrationsUrl, request)
        .then((response) => response.data.integration);
}

export function updateAiIntegration(data: AiIntegration): Promise<Empty> {
    const request: AiIntegrationRequest = { integration: data };
    return axios
        .put<Empty>(`${aiIntegrationsUrl}/${data.id}`, request)
        .then((response) => response.data);
}

export function deleteAiIntegration(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${aiIntegrationsUrl}/${id}`).then((response) => response.data);
}

export function testAiIntegration(data: AiIntegration): Promise<Empty> {
    const request: AiIntegrationRequest = { integration: data };
    return axios
        .post<Empty>(`${aiIntegrationsUrl}/test`, request)
        .then((response) => response.data);
}

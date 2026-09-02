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

type ListAiIntegrationsResponse = {
    integrations: AiIntegration[];
};

export function fetchAiIntegrations(): Promise<{ integrations: AiIntegration[] }> {
    return axios
        .get<ListAiIntegrationsResponse>(aiIntegrationsUrl)
        .then((response) => response.data);
}

export function fetchAiIntegration(id: string): Promise<AiIntegration> {
    return axios.get<AiIntegration>(`${aiIntegrationsUrl}/${id}`).then((response) => response.data);
}

export function createAiIntegration(data: AiIntegration): Promise<AiIntegration> {
    return axios.post<AiIntegration>(aiIntegrationsUrl, data).then((response) => response.data);
}

export function updateAiIntegration(data: AiIntegration): Promise<Empty> {
    return axios
        .put<Empty>(`${aiIntegrationsUrl}/${data.id}`, data)
        .then((response) => response.data);
}

export function deleteAiIntegration(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${aiIntegrationsUrl}/${id}`).then((response) => response.data);
}

export function deleteAiIntegrations(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map(deleteAiIntegration));
}

export function testAiIntegration(data: AiIntegration): Promise<Empty> {
    return axios.post<Empty>(`${aiIntegrationsUrl}/test`, data).then((response) => response.data);
}

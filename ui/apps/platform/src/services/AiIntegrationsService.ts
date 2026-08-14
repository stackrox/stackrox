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

type AiIntegrationsResponse = {
    integrations: AiIntegration[];
};

/**
 * Fetches all AI integrations.
 */
export function fetchAiIntegrations(): Promise<AiIntegration[]> {
    return axios
        .get<AiIntegrationsResponse>(aiIntegrationsUrl)
        .then((response) => response.data.integrations);
}

/**
 * Fetches a single AI integration by ID.
 */
export function fetchAiIntegration(id: string): Promise<AiIntegration> {
    return axios.get<AiIntegration>(`${aiIntegrationsUrl}/${id}`).then((response) => response.data);
}

/**
 * Creates a new AI integration.
 */
export function createAiIntegration(data: AiIntegration): Promise<AiIntegration> {
    return axios.post<AiIntegration>(aiIntegrationsUrl, data).then((response) => response.data);
}

/**
 * Updates an existing AI integration.
 */
export function updateAiIntegration(data: AiIntegration): Promise<AiIntegration> {
    return axios.put<AiIntegration>(aiIntegrationsUrl, data).then((response) => response.data);
}

/**
 * Deletes an AI integration by ID.
 */
export function deleteAiIntegration(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${aiIntegrationsUrl}/${id}`).then((response) => response.data);
}

/**
 * Tests an AI integration connection.
 */
export function testAiIntegration(data: AiIntegration): Promise<Empty> {
    return axios.post<Empty>(`${aiIntegrationsUrl}/test`, data).then((response) => response.data);
}

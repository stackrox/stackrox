import { Traits } from 'types/traits.proto';
import axios from './instance';

import { IntegrationBase, IntegrationOptions } from './IntegrationsService';
import { Empty } from './types';

const notifierIntegrationsUrl = '/v1/notifiers';

const updateIntegrationKey = 'notifier';

// See proto/storage/notifier.proto

export type NotifierIntegrationBase = {
    uiEndpoint: string;
    labelKey: string;
    labelDefault: string;
    traits?: Traits;
} & IntegrationBase;

/*
 * Create integration.
 * The id of argument is empty string and the id of response is assigned by server.
 */
export function createNotifierIntegration(
    integration: NotifierIntegrationBase
): Promise<NotifierIntegrationBase> {
    return axios.post(notifierIntegrationsUrl, integration);
}

/*
 * Read integrations (plural).
 */
export function fetchNotifierIntegrations(): Promise<NotifierIntegrationBase[]> {
    return axios
        .get<{ notifiers: NotifierIntegrationBase[] }>(notifierIntegrationsUrl)
        .then((response) => response?.data?.notifiers ?? []);
}

/*
 * Update integration.
 *
 * Call with options argument if integration has stored credentials, aka password:
 * true to update credentials on the server from the request payload
 * false not to update credentials on the server
 *
 * Call without options argument if integration does not have stored credentials.
 */
export function saveNotifierIntegration(
    integration: NotifierIntegrationBase,
    { updatePassword }: IntegrationOptions = {}
): Promise<Empty> {
    const { id } = integration;

    if (!id) {
        throw new Error('Integration entity must have an id to be saved');
    }

    if (typeof updatePassword === 'boolean') {
        return axios.patch(`${notifierIntegrationsUrl}/${id}`, {
            [updateIntegrationKey]: integration,
            updatePassword,
        });
    }

    return axios.put(`${notifierIntegrationsUrl}/${id}`, integration);
}

/*
 * Test integration.
 *
 * Call with options argument if integration has stored credentials, aka password:
 * true to use credentials in the request payload
 * false to use credentials on the server
 *
 * Call without options argument if integration does not have stored credentials.
 */
export function testNotifierIntegration(
    integration: NotifierIntegrationBase,
    { updatePassword }: IntegrationOptions = {}
): Promise<Empty> {
    if (typeof updatePassword === 'boolean') {
        return axios.post(`${notifierIntegrationsUrl}/test/updated`, {
            [updateIntegrationKey]: integration,
            updatePassword,
        });
    }

    return axios.post(`${notifierIntegrationsUrl}/test`, integration);
}

/*
 * Delete integration (singular).
 */
export function deleteNotifierIntegration(id: string): Promise<Empty> {
    return axios.delete(`${notifierIntegrationsUrl}/${id}`);
}

/*
 * Delete integrations (plural).
 */
export function deleteNotifierIntegrations(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map((id) => deleteNotifierIntegration(id)));
}

import axios from './instance';

import { IntegrationBase, IntegrationOptions } from './IntegrationsService';
import { Empty } from './types';

const backupIntegrationsUrl = '/v1/externalbackups';

const updateIntegrationKey = 'externalBackup';

// See proto/storage/external_backup.proto

export type BackupIntervalType = 'UNSET' | 'DAILY' | 'WEEKLY';

export type BackupSchedule = {
    intervalType: BackupIntervalType;
    hour: number;
    minute: number;
    weekly?: {
        day: number;
    };
};

export type BackupIntegrationBase = {
    schedule: BackupSchedule;
    backupsToKeep: number;
} & IntegrationBase;

/*
 * Create integration.
 * The id of argument is empty string and the id of response is assigned by server.
 */
export function createBackupIntegration(
    integration: BackupIntegrationBase
): Promise<BackupIntegrationBase> {
    return axios.post(backupIntegrationsUrl, integration);
}

/*
 * Read integrations (plural).
 */
export function fetchBackupIntegrations(): Promise<BackupIntegrationBase[]> {
    return axios
        .get<{ externalBackups: BackupIntegrationBase[] }>(backupIntegrationsUrl)
        .then((response) => response?.data?.externalBackups ?? []);
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
export function saveBackupIntegration(
    integration: BackupIntegrationBase,
    { updatePassword }: IntegrationOptions = {}
): Promise<Empty> {
    const { id } = integration;

    if (!id) {
        throw new Error('Integration entity must have an id to be saved');
    }

    if (typeof updatePassword === 'boolean') {
        return axios.patch(`${backupIntegrationsUrl}/${id}`, {
            [updateIntegrationKey]: integration,
            updatePassword,
        });
    }

    return axios.put(`${backupIntegrationsUrl}/${id}`, integration);
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
export function testBackupIntegration(
    integration: BackupIntegrationBase,
    { updatePassword }: IntegrationOptions = {}
): Promise<Empty> {
    if (typeof updatePassword === 'boolean') {
        return axios.post(`${backupIntegrationsUrl}/test/updated`, {
            [updateIntegrationKey]: integration,
            updatePassword,
        });
    }

    return axios.post(`${backupIntegrationsUrl}/test`, integration);
}

/*
 * Delete integration (singular).
 */
export function deleteBackupIntegration(id: string): Promise<Empty> {
    return axios.delete(`${backupIntegrationsUrl}/${id}`);
}

/*
 * Delete integrations (plural).
 */
export function deleteBackupIntegrations(ids: string[]): Promise<Empty[]> {
    return Promise.all(ids.map((id) => deleteBackupIntegration(id)));
}

/*
 * Trigger external backup.
 */
export function triggerBackup(id: string): Promise<Empty> {
    return axios.post(`${backupIntegrationsUrl}/${id}`);
}

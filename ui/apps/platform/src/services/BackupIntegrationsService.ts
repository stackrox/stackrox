import axios from './instance';

import type { IntegrationBase } from './IntegrationsService';
import type { Empty } from './types';

const backupIntegrationsUrl = '/v1/externalbackups';

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
 * Read integrations (plural).
 */
export function fetchBackupIntegrations(): Promise<BackupIntegrationBase[]> {
    return axios
        .get<{ externalBackups: BackupIntegrationBase[] }>(backupIntegrationsUrl)
        .then((response) => response?.data?.externalBackups ?? []);
}

/*
 * Trigger external backup.
 */
export function triggerBackup(id: string): Promise<Empty> {
    return axios.post(`${backupIntegrationsUrl}/${id}`);
}

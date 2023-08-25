import { BackupIntegrationType } from './integration';
import { Schedule } from './schedule.proto';

export type BaseBackupIntegration = {
    id: string;
    name: string;
    type: BackupIntegrationType;
    schedule: Schedule;
    backupsToKeep: number; // int32
};

export type BackupIntegration = GCSBackupIntegration | S3BackupIntegration;

export type GCSConfig = {
    bucket: string;
    // The service account for the storage integration. The server will mask the value of this credential in responses and logs.
    serviceAccount: string; // scrub: always
    objectPrefix: string;
    useWorkloadId: boolean; // scrub: dependent
};

export type GCSBackupIntegration = {
    gcs: GCSConfig;
} & BaseBackupIntegration;

export type S3Config = {
    bucket: string;
    useIam: boolean; // scrub: dependent
    // The access key ID for the storage integration. The server will mask the value of this credential in responses and logs.
    accessKeyId: string; // scrub: always
    // The secret access key for the storage integration. The server will mask the value of this credential in responses and logs.
    secretAccessKey: string; // scrub: always
    region: string;
    objectPrefix: string;
    endpoint: string; // scrub: dependent
};

export type S3BackupIntegration = {
    s3: S3Config;
} & BaseBackupIntegration;

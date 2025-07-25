import type { BackupIntegrationType } from './integration';
import type { Schedule } from './schedule.proto';

export type BaseBackupIntegration = {
    id: string;
    name: string;
    type: BackupIntegrationType;
    schedule: Schedule;
    backupsToKeep: number; // int32
};

export type BackupIntegration =
    | GCSBackupIntegration
    | S3BackupIntegration
    | S3CompatibleBackupIntegration;

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

export type S3URLStyle =
    | 'S3_URL_STYLE_UNSPECIFIED'
    | 'S3_URL_STYLE_VIRTUAL_HOST'
    | 'S3_URL_STYLE_PATH';

export type S3CompatibleConfig = {
    bucket: string;
    // The access key ID for the storage integration. The server will mask the value of this credential in responses and logs.
    accessKeyId: string; // scrub: always
    // The secret access key for the storage integration. The server will mask the value of this credential in responses and logs.
    secretAccessKey: string; // scrub: always
    region: string;
    objectPrefix: string;
    endpoint: string; // scrub: dependent
    // The URL style defines the bucket URL addressing.
    // Virtual-hosted-style buckets are addressed as `https://<bucket>.<endpoint>'
    // while path-style buckets are addressed as `https://<endpoint>/<bucket>`.
    url_style: S3URLStyle;
};

export type S3CompatibleBackupIntegration = {
    s3Compatible: S3CompatibleConfig;
} & BaseBackupIntegration;

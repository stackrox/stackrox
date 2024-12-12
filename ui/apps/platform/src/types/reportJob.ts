import { SlimUser } from 'types/user.proto';

export type Snapshot = {
    reportJobId: string;
    name: string;
    description?: string;
    reportStatus: ReportStatus;
    user: SlimUser;
    isDownloadAvailable: boolean;
};

export type ReportStatus = {
    runState: RunState;
    completedAt: string; // google.protobuf.Timestamp
    errorMsg: string;
    reportRequestType: ReportRequestType;
    reportNotificationMethod: ReportNotificationMethod;
};

export const runStates = {
    WAITING: 'WAITING',
    PREPARING: 'PREPARING',
    GENERATED: 'GENERATED',
    DELIVERED: 'DELIVERED',
    FAILURE: 'FAILURE',
} as const;

export type RunState = (typeof runStates)[keyof typeof runStates];

export type ReportRequestType = 'ON_DEMAND' | 'SCHEDULED';

export type ReportNotificationMethod = 'UNSET' | 'EMAIL' | 'DOWNLOAD';

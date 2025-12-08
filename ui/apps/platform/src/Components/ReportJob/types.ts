import type { ValueOf } from 'utils/type.utils';

export const jobContextTabs = ['CONFIGURATION_DETAILS', 'ALL_REPORT_JOBS'] as const;

export type JobContextTab = (typeof jobContextTabs)[number];

export const reportJobStatuses = {
    WAITING: 'WAITING',
    PREPARING: 'PREPARING',
    DOWNLOAD_GENERATED: 'DOWNLOAD_GENERATED',
    PARTIAL_SCAN_ERROR_DOWNLOAD: 'PARTIAL_SCAN_ERROR_DOWNLOAD',
    EMAIL_DELIVERED: 'EMAIL_DELIVERED',
    PARTIAL_SCAN_ERROR_EMAIL: 'PARTIAL_SCAN_ERROR_EMAIL',
    ERROR: 'ERROR',
} as const;

export type ReportJobStatus = ValueOf<typeof reportJobStatuses>;

export const reportJobStatusLabels: Record<ReportJobStatus, string> = {
    WAITING: 'Waiting',
    PREPARING: 'Preparing',
    DOWNLOAD_GENERATED: 'Report ready for download',
    PARTIAL_SCAN_ERROR_DOWNLOAD: 'Partial report ready for download',
    EMAIL_DELIVERED: 'Report successfully sent',
    PARTIAL_SCAN_ERROR_EMAIL: 'Partial report successfully sent',
    ERROR: 'Report failed to generate',
};

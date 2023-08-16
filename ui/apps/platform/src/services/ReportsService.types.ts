import { SlimUser } from 'types/user.proto';
import { VulnerabilitySeverity } from '../types/cve.proto';

// Report configuration types

export type ReportConfiguration = {
    id: string;
    name: string;
    description: string;
    type: ReportType;
    vulnReportFilters: VulnerabilityReportFilters;
    notifiers: NotifierConfiguration[];
    schedule: Schedule | null;
    resourceScope: ResourceScope;
};

export type ReportType = 'VULNERABILITY';

export type VulnerabilityReportFiltersBase = {
    fixability: Fixability;
    severities: VulnerabilitySeverity[];
    imageTypes: ImageType[];
};

export type VulnerabilityReportFilters =
    | (VulnerabilityReportFiltersBase & {
          allVuln: boolean;
      })
    | (VulnerabilityReportFiltersBase & {
          sinceLastSentScheduledReport: boolean;
      })
    | (VulnerabilityReportFiltersBase & {
          sinceStartDate: string; // in the format of google.protobuf.Timestamp};
      });

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE';

export const imageTypes = ['DEPLOYED', 'WATCHED'] as const;

export type ImageType = (typeof imageTypes)[number];

export type NotifierConfiguration = {
    emailConfig: {
        notifierId: string;
        mailingLists: string[];
    };
    notifierName: string;
};

export type Schedule =
    | {
          intervalType: 'WEEKLY';
          hour: number;
          minute: number;
          daysOfWeek: DaysOfWeek;
      }
    | {
          intervalType: 'MONTHLY';
          hour: number;
          minute: number;
          daysOfMonth: DaysOfMonth;
      };

export const intervalTypes = ['WEEKLY', 'MONTHLY'] as const;

export type IntervalType = (typeof intervalTypes)[number];

export type Interval = DaysOfWeek | DaysOfMonth;

// Sunday = 0, Monday = 1, .... Saturday =  6
export type DaysOfWeek = {
    days: number[]; // int32
};

// 1 for 1st, 2 for 2nd .... 31 for 31st
export type DaysOfMonth = {
    days: number[]; // int32
};

export type ResourceScope = {
    collectionScope: {
        collectionId: string;
        collectionName: string;
    };
};

// Report status types

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
    SUCCESS: 'SUCCESS',
    FAILURE: 'FAILURE',
} as const;

export type RunState = (typeof runStates)[keyof typeof runStates];

export type ReportRequestType = 'ON_DEMAND' | 'SCHEDULED';

export type ReportNotificationMethod = 'UNSET' | 'EMAIL' | 'DOWNLOAD';

// Report history

export type ReportHistoryResponse = {
    reportSnapshots: ReportSnapshot[];
};

export type ReportSnapshot = {
    reportConfigId: string;
    reportJobId: string;
    name: string;
    description: string;
    vulnReportFilters: VulnerabilityReportFilters;
    collectionSnapshot: CollectionSnapshot;
    schedule: Schedule;
    reportStatus: ReportStatus;
    notifiers: NotifierConfiguration[];
    user: SlimUser;
    isDownloadAvailable: boolean;
};

export type CollectionSnapshot = {
    id: string;
    name: string;
};

// Misc types

export type RunReportResponse = {
    reportConfigId: string;
    reportId: string;
};

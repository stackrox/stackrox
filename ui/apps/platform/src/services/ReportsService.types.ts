import { VulnerabilitySeverity } from '../types/cve.proto';

// Report configuration types

export type ReportConfiguration = {
    id: string;
    name: string;
    description: string;
    type: ReportType;
    vulnReportFilters: VulnerabilityReportFilters;
    notifiers: NotifierConfiguration[];
    schedule: Schedule;
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
          lastSuccessfulReport: boolean;
      })
    | (VulnerabilityReportFiltersBase & {
          startDate: string; // in the format of google.protobuf.Timestamp};
      });

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE';

export type ImageType = 'DEPLOYED' | 'WATCHED';

export type NotifierConfiguration = {
    emailConfig: {
        notifierId: string;
        mailingLists: string[];
    };
    notifierName: string;
};

type ScheduleBase = {
    intervalType: IntervalType;
    hour: number;
    minute: number;
};

export type Schedule =
    | (ScheduleBase & {
          daysOfWeek: DaysOfWeek;
      })
    | (ScheduleBase & {
          daysOfMonth: DaysOfMonth;
      });

export type IntervalType = 'WEEKLY' | 'MONTHLY';

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

export type RunState = 'WAITING' | 'PREPARING' | 'SUCCESS' | 'FAILURE';

export type ReportRequestType = 'ON_DEMAND' | 'SCHEDULED';

export type ReportNotificationMethod = 'UNSET' | 'EMAIL' | 'DOWNLOAD';

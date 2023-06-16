import { VulnerabilitySeverity } from './cve.proto';

export type ReportType = 'VULNERABILITY';

export type ReportConfiguration = {
    id: string;
    name: string;
    description: string;
    type: ReportType;
    vulnReportFilters: VulnerabilityReportFilters;
    emailConfig: EmailNotifierConfiguration;
    schedule: Schedule;
    resourceScope: ResourceScope;
};

type VulnerabilityReportFiltersBase = {
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

export type EmailNotifierConfiguration = {
    notifierId: string;
    mailingLists: string[];
};

type ScheduleBase = {
    intervalType: IntervalType;
    hour: number;
    minute: number;
};

export type Schedule =
    | (ScheduleBase & {
          weekly: WeeklyInterval;
      })
    | (ScheduleBase & {
          daysOfWeek: DaysOfWeek;
      })
    | (ScheduleBase & {
          daysOfMonth: DaysOfMonth;
      });

export type IntervalType = 'UNSET' | 'DAILY' | 'WEEKLY' | 'MONTHLY';

export type Interval = DaysOfWeek | DaysOfMonth;

// TODO - Note that the types of `days` below are not exact, the API returns `number[]`, but the UI converts
// them to strings. This doesn't seem to be a problem, but it's worth noting.

export type WeeklyInterval = {
    day: string; // int32
};

// Sunday = 0, Monday = 1, .... Saturday =  6
export type DaysOfWeek = {
    days: string[]; // int32
};

// 1 for 1st, 2 for 2nd .... 31 for 31st
export type DaysOfMonth = {
    days: string[]; // int32
};

export type ResourceScope = {
    collectionScope: {
        collectionId: string;
        collectionName: string;
    };
};

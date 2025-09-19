import type { Snapshot } from 'types/reportJob';
import type { VulnerabilitySeverity } from '../types/cve.proto';

// Core report types

export type ReportType = 'VULNERABILITY';

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

// Vulnerability report filters

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE';

export const imageTypes = ['DEPLOYED', 'WATCHED'] as const;
export type ImageType = (typeof imageTypes)[number];

export type VulnerabilityReportFiltersBase = {
    fixability: Fixability;
    severities: VulnerabilitySeverity[];
    imageTypes: ImageType[];
    includeAdvisory: boolean;
    includeEpssProbability: boolean;
    includeKnownExploit: boolean;
    includeNvdCvss: boolean;
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

export type ViewBasedVulnerabilityReportFilters = {
    query: string;
};

// Scheduling types

export const intervalTypes = ['WEEKLY', 'MONTHLY'] as const;
export type IntervalType = (typeof intervalTypes)[number];

// Sunday = 0, Monday = 1, .... Saturday =  6
export type DaysOfWeek = {
    days: number[]; // int32
};

// 1 for 1st, 2 for 2nd .... 31 for 31st
export type DaysOfMonth = {
    days: number[]; // int32
};

export type Interval = DaysOfWeek | DaysOfMonth;

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

// Notification types

export type NotifierConfiguration = {
    emailConfig: {
        notifierId: string;
        mailingLists: string[];
        customSubject: string;
        customBody: string;
    };
    notifierName: string;
};

// Resource scope types

export type ResourceScope = {
    collectionScope: {
        collectionId: string;
        collectionName: string;
    };
};

export type CollectionSnapshot = {
    id: string;
    name: string;
};

// Report history types

export type ReportHistoryResponse = {
    reportSnapshots: ReportSnapshot[];
};

export type ViewBasedReportSnapshot = Snapshot & {
    viewBasedVulnReportFilters: ViewBasedVulnerabilityReportFilters;
    requestName: string;
    areaOfConcern: string;
};

export type ConfiguredReportSnapshot = Snapshot & {
    reportConfigId: string;
    vulnReportFilters: VulnerabilityReportFilters;
    collectionSnapshot: CollectionSnapshot;
    schedule: Schedule | null;
    notifiers: NotifierConfiguration[];
};

export type ReportSnapshot = ConfiguredReportSnapshot | ViewBasedReportSnapshot;

// Type guard functions

export function isViewBasedReportSnapshot(
    snapshot: ReportSnapshot
): snapshot is ViewBasedReportSnapshot {
    return 'viewBasedVulnReportFilters' in snapshot;
}

export function isConfiguredReportSnapshot(
    snapshot: ReportSnapshot
): snapshot is ConfiguredReportSnapshot {
    return 'reportConfigId' in snapshot;
}

// API request/response types

export type RunReportResponse = {
    reportConfigId: string;
    reportId: string;
};

export type ReportRequestViewBased = {
    type: ReportType;
    viewBasedVulnReportFilters: ViewBasedVulnerabilityReportFilters;
    areaOfConcern: string;
};

export type RunReportResponseViewBased = {
    reportID: string;
    requestName: string;
};

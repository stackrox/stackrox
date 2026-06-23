import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { Snapshot } from 'types/reportJob';

// Core report types

export type ReportType = 'VULNERABILITY' | 'NODE_VULNERABILITY';

export type ReportConfigurationBase = {
    id: string;
    name: string;
    description: string;
    notifiers: NotifierConfiguration[];
    schedule: Schedule | null;
};

export type GenericVulnerabilityReportConfiguration = {
    type: 'NODE_VULNERABILITY';
    vulnReportFilters: GenericVulnerabilityReportFilters;
    resourceScope: {
        entityScope: EntityScope;
    };
} & ReportConfigurationBase;

// TODO temporary alias to limit changed files that might be superseded later anyway
export type ReportConfiguration = ImageVulnerabilityReportConfiguration;

// After we remove ForCollection, ForEntity becomes the report configuration.
export type ImageVulnerabilityReportConfiguration =
    | ImageVulnerabilityReportConfigurationForEntity
    | ImageVulnerabilityReportConfigurationForCollection;

export type ImageVulnerabilityReportConfigurationForEntity = {
    type: 'VULNERABILITY';
    vulnReportFilters: ImageVulnerabilityReportFiltersForEntity;
    resourceScope: {
        entityScope: EntityScope;
    };
} & ReportConfigurationBase;

export type ImageVulnerabilityReportConfigurationForCollection = {
    type: 'VULNERABILITY';
    vulnReportFilters: ImageVulnerabilityReportFiltersForCollection;
    resourceScope: {
        collectionScope: CollectionScope;
    };
} & ReportConfigurationBase;

// Vulnerability report filters

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE';

export const imageTypes = ['DEPLOYED', 'WATCHED'] as const;
export type ImageType = (typeof imageTypes)[number];

export type ImageVulnerabilityReportFiltersForEntity = {
    imageTypes: ImageType[];
    query: string;
} & CvesSince;

export type ImageVulnerabilityReportFiltersForCollection = {
    fixability: Fixability;
    severities: VulnerabilitySeverity[];
    imageTypes: ImageType[];
} & CvesSince;

export type CvesSince =
    | {
          allVuln: boolean;
      }
    | {
          sinceLastSentScheduledReport: boolean;
      }
    | {
          sinceStartDate: string; // in the format of google.protobuf.Timestamp};
      };

export type GenericVulnerabilityReportFilters = {
    query: string;
} & CvesSince;

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
    collectionScope: CollectionScope;
};

export type CollectionScope = {
    collectionId: string;
    collectionName: string;
};

export type EntityScope = {
    rules: EntityScopeRule[];
};

export type MatchType = 'EXACT' | 'REGEX';

export type RuleValue = {
    value: string;
    matchType: MatchType;
};

export type EntityScopeRule = {
    entity: ScopeEntity;
    field: ScopeField;
    values: RuleValue[];
};

export type ScopeEntity =
    | 'SCOPE_ENTITY_UNSET'
    | 'SCOPE_ENTITY_DEPLOYMENT'
    | 'SCOPE_ENTITY_NAMESPACE'
    | 'SCOPE_ENTITY_CLUSTER';

export type ScopeField =
    | 'FIELD_UNSET'
    | 'FIELD_ID'
    | 'FIELD_NAME'
    | 'FIELD_LABEL'
    | 'FIELD_ANNOTATION';

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
    areaOfConcern: string;
};

// TODO temporary disjunction until snamshot has type property.
type VulnerabilityReportFilters =
    | GenericVulnerabilityReportFilters
    | ImageVulnerabilityReportConfigurationForCollection
    | ImageVulnerabilityReportConfigurationForEntity;

// TODO distinguish configured versus view-based instead of combining them.
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

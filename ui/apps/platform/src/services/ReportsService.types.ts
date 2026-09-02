import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { Snapshot } from 'types/reportJob';

// Core report types

export type ReportType = 'VULNERABILITY' | 'NODE_VULNERABILITY';

export type ReportConfigurationBase = {
    id: string;
    name: string;
    description: string;
    notifiers: NotifierConfiguration[];
    schedule: ReportSchedule | null;
};

// Not exactly 1:1 with the proto, which uses a oneOf that technically allows mismatched
// type/filters (e.g. node type with image filters). Backend confirmed that can't happen, so
// we pin `type` to its matching filters to simplify the typing.
export type NodeVulnerabilityReportConfiguration = {
    type: 'NODE_VULNERABILITY';
} & ReportConfigurationBase &
    NodeVulnerabilityReportFiltersConfiguration &
    NodeVulnerabilityReportResourcesConfiguration;

// Types of views and steps with only properties that they need to know:
export type NodeVulnerabilityReportFiltersConfiguration = {
    nodeVulnReportFilters: NodeVulnerabilityReportFilters;
};

export type NodeVulnerabilityReportResourcesConfiguration = {
    resourceScope: NodeVulnerabilityReportResourceScope;
};

// Types of properties:
export type NodeVulnerabilityReportFilters = {
    allVuln: boolean;
    query: string;
};

export type NodeVulnerabilityReportResourceScope = {
    entityScope: EntityScope; // Cluster
};

// Draft of future configuration type.
export type VirtualMachineVulnerabilityReportConfiguration = {
    type: 'VIRTUAL_MACHINE_VULNERABILITY';
} & ReportConfigurationBase &
    VirtualMachineVulnerabilityReportFiltersConfiguration &
    VirtualMachineVulnerabilityReportResourcesConfiguration;

// Types of views and steps with only properties that they need to know:
export type VirtualMachineVulnerabilityReportFiltersConfiguration = {
    virtualMachineVulnReportFilters: VirtualMachineVulnerabilityReportFilters;
};

export type VirtualMachineVulnerabilityReportResourcesConfiguration = {
    resourceScope: VirtualMachineVulnerabilityReportResourceScope;
};

// Types of properties:
export type VirtualMachineVulnerabilityReportFilters = {
    allVuln: boolean;
    query: string;
};

export type VirtualMachineVulnerabilityReportResourceScope = {
    entityScope: EntityScope; // Cluster and Namespace
};

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

// Types of views and steps with only properties that they need to know:
export type ImageVulnerabilityReportResourcesConfiguration = {
    resourceScope: ImageVulnerabilityReportResourceScope;
    vulnReportFilters: {
        imageTypes: ImageType[]; // more closely related to resources although in vulnReportFilters
    };
};

export type ImageVulnerabilityReportResourceScope =
    | {
          collectionScope: CollectionScope;
      }
    | {
          entityScope: EntityScope;
      };

// Types because Resources and Filters for view and wizard combine data properties.
export type ImageVulnerabilityReportFiltersConfiguration = {
    vulnReportFilters: ImageVulnerabilityReportFiltersWithoutCvesSince & CvesSince;
};

export type ImageVulnerabilityReportFiltersWithoutCvesSince =
    | ImageVulnerabilityReportFiltersForEntityWithoutCvesSince
    | ImageVulnerabilityReportFiltersForCollectionWithoutCvesSince;

export type ImageVulnerabilityReportFiltersConfigurationForEntity = {
    vulnReportFilters: ImageVulnerabilityReportFiltersForEntityWithoutCvesSince & CvesSince;
};

export type ImageVulnerabilityReportFiltersForEntityWithoutCvesSince = {
    query: string;
};

export type ImageVulnerabilityReportFiltersConfigurationForCollection = {
    vulnReportFilters: ImageVulnerabilityReportFiltersForCollectionWithoutCvesSince & CvesSince;
};

export type ImageVulnerabilityReportFiltersForCollectionWithoutCvesSince = {
    fixability: Fixability;
    severities: VulnerabilitySeverity[];
};

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

export type ReportSchedule =
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
      }
    | {
          intervalType: 'DAILY';
          hour: number;
          minute: number;
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

export type ResourceScope =
    | {
          collectionScope: CollectionScope;
      }
    | {
          entityScope: EntityScope;
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
    type: 'VULNERABILITY';
    viewBasedVulnReportFilters: ViewBasedVulnerabilityReportFilters;
    areaOfConcern: string;
};

export type NodeViewBasedReportSnapshot = Snapshot & {
    type: 'NODE_VULNERABILITY';
    nodeVulnReportFilters: NodeVulnerabilityReportFilters;
    areaOfConcern: string;
};

type VulnerabilityReportFilters =
    | NodeVulnerabilityReportFilters
    | ImageVulnerabilityReportFiltersForCollection
    | ImageVulnerabilityReportFiltersForEntity;

// TODO distinguish configured versus view-based instead of combining them.
export type ConfiguredReportSnapshot = Snapshot & {
    type: ReportType;
    reportConfigId: string;
    vulnReportFilters: VulnerabilityReportFilters;
    collectionSnapshot: CollectionSnapshot;
    schedule: ReportSchedule | null;
    notifiers: NotifierConfiguration[];
};

export type ReportSnapshot =
    | ConfiguredReportSnapshot
    | ViewBasedReportSnapshot
    | NodeViewBasedReportSnapshot;

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

export type ImageReportRequestViewBased = {
    type: 'VULNERABILITY';
    viewBasedVulnReportFilters: ViewBasedVulnerabilityReportFilters;
    areaOfConcern: string;
};

export type NodeReportRequestViewBased = {
    type: 'NODE_VULNERABILITY';
    // Although backend proto has NodeVulnerabilityReportFilters type,
    // only query is relevant for view-based report.
    nodeVulnReportFilters: ViewBasedVulnerabilityReportFilters;
    areaOfConcern: string; // 'Nodes'
};

export type ReportRequestViewBased = ImageReportRequestViewBased | NodeReportRequestViewBased;

export type RunReportResponseViewBased = {
    reportID: string;
    requestName: string;
};

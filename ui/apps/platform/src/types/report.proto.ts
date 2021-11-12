import type { VulnerabilitySeverity } from 'messages/common';

export type ReportConfiguration = {
    id: string;
    name: string;
    description: string;
    type: ReportType;
    filter: VulnerabilityReportFilters;
    scopeId: string;
    notifierConfig: {
        emailConfig: EmailNotifierConfiguration;
    };
    schedule: Schedule;
    runStatus: ReportLastRunStatus;
};

// TODO: COMPLIANCE type is for a future feature, currently a comment in proto file
export type ReportType = 'VULNERABILITY' | 'COMPLIANCE';

export type ReportLastRunStatus = {
    reportStatus: RunStatus;
    lastTimeRun: string; // ISO 8601 date string
    errorMsg: string;
};

export type RunStatus = 'SUCCESS' | 'FAILURE';

export type VulnerabilityReportFilters = {
    fixability: Fixability;
    sinceLastReport: boolean;
    severities: VulnerabilitySeverity[];
};

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE';

export type EmailNotifierConfiguration = {
    notifierId: string;
    mailingLists: string[];
};

export type Schedule = {
    intervalType: IntervalType;
    hour: number;
    minute: number;
    interval: Interval;
};

export type IntervalType = 'UNSET' | 'DAILY' | 'WEEKLY' | 'EVERY_TWO_WEEKS' | 'MONTHLY';

export type Interval = WeeklyInterval | DaysOfWeek | DaysOfMonth;

// Note: This field will be unused for vuln mgmt reporting. It is currently
// in use for scheduled S3 backups. With an appropriate migration, it can be
// deprecated, and DaysOfWeek can be used instead.
export type WeeklyInterval = {
    day: number; // int32
};

// Sunday = 0, Monday = 1, .... Saturday =  6
export type DaysOfWeek = {
    day: number; // int32
};
// Only 1st and 15th of the month allowed for vuln report scheduling (API validations will be done)
export type DaysOfMonth = {
    days: number[]; // int32
};

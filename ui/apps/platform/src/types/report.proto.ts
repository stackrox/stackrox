import { VulnerabilitySeverity } from './cve.proto';

export type ReportConfiguration = {
    id: string;
    name: string;
    description: string;
    type: ReportType;
    vulnReportFilters: VulnerabilityReportFilters;
    scopeId: string;
    emailConfig: EmailNotifierConfiguration;
    schedule: Schedule;
    runStatus?: ReportLastRunStatus;
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

export type Fixability = 'BOTH' | 'FIXABLE' | 'NOT_FIXABLE' | 'UNSET';

export type EmailNotifierConfiguration = {
    notifierId: string;
    mailingLists: string[];
};

export type Schedule = {
    intervalType: IntervalType;
    hour: number;
    minute: number;
    daysOfWeek?: DaysOfWeek;
    daysOfMonth?: DaysOfMonth;
};

export type IntervalType = 'UNSET' | 'DAILY' | 'WEEKLY' | 'MONTHLY';

export type Interval = DaysOfWeek | DaysOfMonth;

// Sunday = 0, Monday = 1, .... Saturday =  6
export type DaysOfWeek = {
    days: string[]; // int32
};
// Only 1st and 15th of the month allowed for vuln report scheduling (API validations will be done)
export type DaysOfMonth = {
    days: string[]; // int32
};

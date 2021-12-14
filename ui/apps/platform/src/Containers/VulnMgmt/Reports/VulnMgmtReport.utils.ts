import { ReportConfiguration } from 'types/report.proto';
import { ExtendedPageAction } from 'utils/queryStringUtils';

export type VulnMgmtReportQueryObject = {
    action?: ExtendedPageAction;
};

export const emptyReportValues: ReportConfiguration = {
    id: '',
    name: '',
    description: '',
    type: 'VULNERABILITY',
    vulnReportFilters: {
        fixability: 'BOTH',
        sinceLastReport: false,
        severities: [],
    },
    scopeId: '',
    notifierConfig: {
        emailConfig: {
            notifierId: '',
            mailingLists: [],
        },
    },
    schedule: {
        intervalType: 'WEEKLY',
        hour: 0,
        minute: 0,
        interval: {
            days: [],
        },
    },
};

import { FixabilityLabelKey } from 'constants/reportConstants';
import { ReportConfiguration, Fixability } from 'types/report.proto';
import { ExtendedPageAction } from 'utils/queryStringUtils';

export type VulnMgmtReportQueryObject = {
    action?: ExtendedPageAction;
    s?: Record<string, string>;
    p?: string; // really a page number, but all URL params are parsed as strings
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
    emailConfig: {
        notifierId: '',
        mailingLists: [],
    },
    schedule: {
        intervalType: 'WEEKLY',
        hour: 0,
        minute: 0,
        daysOfWeek: {
            days: [],
        },
    },
};

export function getMappedFixability(fixability: Fixability): FixabilityLabelKey[] {
    if (fixability === 'BOTH') {
        return ['FIXABLE', 'NOT_FIXABLE'];
    }
    if (fixability === 'FIXABLE') {
        return ['FIXABLE'];
    }
    if (fixability === 'NOT_FIXABLE') {
        return ['NOT_FIXABLE'];
    }
    return [];
}

export function getFixabilityConstantFromMap(fixabilityMap: FixabilityLabelKey[]): Fixability {
    if (fixabilityMap.includes('FIXABLE') && fixabilityMap.includes('NOT_FIXABLE')) {
        return 'BOTH';
    }
    if (fixabilityMap.includes('FIXABLE')) {
        return 'FIXABLE';
    }
    if (fixabilityMap.includes('NOT_FIXABLE')) {
        return 'NOT_FIXABLE';
    }
    return 'UNSET';
}

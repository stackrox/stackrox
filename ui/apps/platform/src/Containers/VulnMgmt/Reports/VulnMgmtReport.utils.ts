import { FixabilityLabelKey } from 'constants/reportConstants';
import { HasReadAccess, HasReadWriteAccess } from 'hooks/usePermissions';
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
        severities: [
            'CRITICAL_VULNERABILITY_SEVERITY',
            'IMPORTANT_VULNERABILITY_SEVERITY',
            'MODERATE_VULNERABILITY_SEVERITY',
            'LOW_VULNERABILITY_SEVERITY',
        ],
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

// Single source of truth for conditional rendering in container.
export function getWriteAccessForReport({
    hasReadAccess,
    hasReadWriteAccess,
}: {
    hasReadAccess: HasReadAccess;
    hasReadWriteAccess: HasReadWriteAccess;
}) {
    // Run report now action requires Access and Image resources.
    // Access seems like a leftover from access scope as report scope.
    return (
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Access') &&
        hasReadAccess('Image')
    );
}

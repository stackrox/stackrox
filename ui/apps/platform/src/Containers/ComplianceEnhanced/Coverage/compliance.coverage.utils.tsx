import React from 'react';
import type { ReactElement } from 'react';
import { Icon } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';
import {
    BanIcon,
    CheckCircleIcon,
    ExclamationCircleIcon,
    ExclamationTriangleIcon,
    ResourcesEmptyIcon,
    SecurityIcon,
    UnknownIcon,
    WrenchIcon,
} from '@patternfly/react-icons';

import type { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import type { SearchFilter } from 'types/search';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { getPercentage } from 'utils/mathUtils';

import { SCAN_CONFIG_NAME_QUERY } from '../compliance.constants';
import {
    FAILING_LABEL_COLOR,
    FAILING_VAR_COLOR,
    MANUAL_LABEL_COLOR,
    MANUAL_VAR_COLOR,
    OTHER_LABEL_COLOR,
    OTHER_VAR_COLOR,
    PASSING_LABEL_COLOR,
    PASSING_VAR_COLOR,
} from './compliance.coverage.constants';

export type ClusterStatusObject = {
    icon: ReactElement;
    statusText: string;
    tooltipText: string | null; // null if tooltip text is redundant with statusText
    color: LabelProps['color'];
};

export const ComplianceStatus = {
    SUCCESS: 'success',
    WARNING: 'warning',
    DANGER: 'danger',
} as const;

export type ComplianceStatus = (typeof ComplianceStatus)[keyof typeof ComplianceStatus];

export function getStatusCounts(checkStats: ComplianceCheckStatusCount[]): {
    passCount: number;
    failCount: number;
    manualCount: number;
    otherCount: number;
    totalCount: number;
} {
    let passCount = 0;
    let failCount = 0;
    let manualCount = 0;
    let otherCount = 0;
    let totalCount = 0;

    checkStats.forEach((statusInfo) => {
        totalCount += statusInfo.count;
        switch (statusInfo.status) {
            case 'PASS':
                passCount += statusInfo.count;
                break;
            case 'FAIL':
                failCount += statusInfo.count;
                break;
            case 'MANUAL':
                manualCount += statusInfo.count;
                break;
            default:
                otherCount += statusInfo.count;
        }
    });

    return { passCount, failCount, manualCount, otherCount, totalCount };
}

export function getStatusPercentages(checkStats: ComplianceCheckStatusCount[]): {
    passPercentage: number;
    failPercentage: number;
    manualPercentage: number;
    otherPercentage: number;
} {
    const { passCount, failCount, manualCount, otherCount, totalCount } =
        getStatusCounts(checkStats);

    const passPercentage = getPercentage(passCount, totalCount);
    const failPercentage = getPercentage(failCount, totalCount);
    const manualPercentage = getPercentage(manualCount, totalCount);
    const otherPercentage = getPercentage(otherCount, totalCount);

    return {
        passPercentage,
        failPercentage,
        manualPercentage,
        otherPercentage,
    };
}

export function sortCheckStats(items: ComplianceCheckStatusCount[]): ComplianceCheckStatusCount[] {
    const order: ComplianceCheckStatus[] = [
        'PASS',
        'FAIL',
        'MANUAL',
        'ERROR',
        'INFO',
        'NOT_APPLICABLE',
        'INCONSISTENT',
        'UNSET_CHECK_STATUS',
    ];
    return [...items].sort((a, b) => {
        return order.indexOf(a.status) - order.indexOf(b.status);
    });
}

const statusIconTextMap: Record<ComplianceCheckStatus, ClusterStatusObject> = {
    PASS: {
        icon: (
            <Icon>
                <CheckCircleIcon color={PASSING_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Pass',
        tooltipText: null,
        color: PASSING_LABEL_COLOR,
    },
    FAIL: {
        icon: (
            <Icon>
                <SecurityIcon color={FAILING_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Fail',
        tooltipText: null,
        color: FAILING_LABEL_COLOR,
    },
    ERROR: {
        icon: (
            <Icon>
                <ExclamationTriangleIcon color={OTHER_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Error',
        tooltipText: 'Check ran successfully, but could not complete',
        color: OTHER_LABEL_COLOR,
    },
    INFO: {
        icon: (
            <Icon>
                <ExclamationCircleIcon color={OTHER_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Info',
        tooltipText:
            'Check was successful and found something not severe enough to be considered an error',
        color: OTHER_LABEL_COLOR,
    },
    MANUAL: {
        icon: (
            <Icon>
                <WrenchIcon color={MANUAL_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Manual',
        tooltipText:
            'Manual check requires additional organizational or technical knowledge that is not automatable',
        color: MANUAL_LABEL_COLOR,
    },
    NOT_APPLICABLE: {
        icon: (
            <Icon>
                <BanIcon color={OTHER_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Not Applicable',
        tooltipText: 'Check did not run as it is not applicable',
        color: OTHER_LABEL_COLOR,
    },
    INCONSISTENT: {
        icon: (
            <Icon>
                <UnknownIcon color={OTHER_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Inconsistent',
        tooltipText: 'Different nodes report different results',
        color: OTHER_LABEL_COLOR,
    },
    UNSET_CHECK_STATUS: {
        icon: (
            <Icon>
                <ResourcesEmptyIcon color={OTHER_VAR_COLOR} />
            </Icon>
        ),
        statusText: 'Unset',
        tooltipText: '',
        color: OTHER_LABEL_COLOR,
    },
};

export function getClusterResultsStatusObject(status: ComplianceCheckStatus): ClusterStatusObject {
    return statusIconTextMap[status];
}

export function createScanConfigFilter(selectedScanConfigName: string | undefined) {
    return selectedScanConfigName ? { [SCAN_CONFIG_NAME_QUERY]: selectedScanConfigName } : {};
}

export function combineSearchFilterWithScanConfig(
    searchFilter: SearchFilter,
    selectedScanConfigName: string | undefined
): SearchFilter {
    return {
        ...searchFilter,
        ...createScanConfigFilter(selectedScanConfigName),
    };
}

export function isScanConfigurationDisabled(
    config: ComplianceScanConfigurationStatus,
    disabledCriteria: { profileName?: string; clusterId?: string } = {}
): boolean {
    const { profileName, clusterId } = disabledCriteria;

    if (profileName && !config.scanConfig.profiles.includes(profileName)) {
        return true;
    }

    if (clusterId && !config.clusterStatus.some((cluster) => cluster.clusterId === clusterId)) {
        return true;
    }

    return false;
}

export function getTimeDifferenceAsPhrase(iso8601: string | null, date: Date) {
    return iso8601 ? getDistanceStrictAsPhrase(iso8601, date) : 'Scanning now';
}

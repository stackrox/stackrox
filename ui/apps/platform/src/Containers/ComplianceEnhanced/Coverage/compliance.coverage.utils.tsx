import React, { ReactElement } from 'react';
import { Icon, LabelProps } from '@patternfly/react-core';
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

import { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import { SearchFilter } from 'types/search';

import { SCAN_CONFIG_NAME_QUERY } from '../compliance.constants';

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

export function calculateCompliancePercentage(passCount: number, totalCount: number): number {
    return totalCount > 0 ? Math.round((passCount / totalCount) * 100) : 0;
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
                <CheckCircleIcon color="var(--pf-v5-global--primary-color--100)" />
            </Icon>
        ),
        statusText: 'Pass',
        tooltipText: null,
        color: 'blue',
    },
    FAIL: {
        icon: (
            <Icon>
                <SecurityIcon color="var(--pf-v5-global--danger-color--100)" />
            </Icon>
        ),
        statusText: 'Fail',
        tooltipText: null,
        color: 'red',
    },
    ERROR: {
        icon: (
            <Icon>
                <ExclamationTriangleIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Error',
        tooltipText: 'Check ran successfully, but could not complete',
        color: 'grey',
    },
    INFO: {
        icon: (
            <Icon>
                <ExclamationCircleIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Info',
        tooltipText:
            'Check was successful and found something not severe enough to be considered an error',
        color: 'grey',
    },
    MANUAL: {
        icon: (
            <Icon>
                <WrenchIcon color="var(--pf-v5-global--warning-color--100)" />
            </Icon>
        ),
        statusText: 'Manual',
        tooltipText:
            'Manual check requires additional organizational or technical knowledge that is not automatable',
        color: 'gold',
    },
    NOT_APPLICABLE: {
        icon: (
            <Icon>
                <BanIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Not Applicable',
        tooltipText: 'Check did not run as it is not applicable',
        color: 'grey',
    },
    INCONSISTENT: {
        icon: (
            <Icon>
                <UnknownIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Inconsistent',
        tooltipText: 'Different nodes report different results',
        color: 'grey',
    },
    UNSET_CHECK_STATUS: {
        icon: (
            <Icon>
                <ResourcesEmptyIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Unset',
        tooltipText: '',
        color: 'grey',
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

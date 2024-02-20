import React, { ReactElement } from 'react';
import { LabelProps } from '@patternfly/react-core';
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

import {
    ComplianceCheckStatus,
    ComplianceCheckStatusCount,
} from 'services/ComplianceEnhancedService';

// Thresholds for compliance status
const DANGER_THRESHOLD = 50;
const WARNING_THRESHOLD = 75;

type LabelColor = LabelProps['color'];

type ClusterStatusObject = {
    icon: ReactElement;
    statusText: string;
    tooltipText: string;
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
    otherCount: number;
    totalCount: number;
} {
    let passCount = 0;
    let failCount = 0;
    let otherCount = 0;
    let totalCount = 0;

    checkStats.forEach((statusInfo) => {
        totalCount += statusInfo.count;
        switch (statusInfo.status) {
            case ComplianceCheckStatus.PASS:
                passCount += statusInfo.count;
                break;
            case ComplianceCheckStatus.FAIL:
                failCount += statusInfo.count;
                break;
            default:
                otherCount += statusInfo.count;
        }
    });

    return { passCount, failCount, otherCount, totalCount };
}

export function calculateCompliancePercentage(passCount: number, totalCount: number): number {
    return totalCount > 0 ? Math.round((passCount / totalCount) * 100) : 0;
}

function getComplianceStatus(passPercentage: number): ComplianceStatus {
    let status: ComplianceStatus = ComplianceStatus.SUCCESS;

    if (passPercentage < DANGER_THRESHOLD) {
        status = ComplianceStatus.DANGER;
    } else if (passPercentage < WARNING_THRESHOLD) {
        status = ComplianceStatus.WARNING;
    }

    return status;
}

export function getCompliancePfClassName(passPercentage: number): string {
    const status = getComplianceStatus(passPercentage);

    if (status === ComplianceStatus.DANGER) {
        return 'pf-m-danger';
    }
    if (status === ComplianceStatus.WARNING) {
        return 'pf-m-warning';
    }
    return '';
}

export function getComplianceLabelGroupColor(
    passPercentage: number | undefined
): LabelColor | undefined {
    if (passPercentage === undefined) {
        return undefined;
    }

    const status = getComplianceStatus(passPercentage);

    if (status === ComplianceStatus.DANGER) {
        return 'red';
    }
    if (status === ComplianceStatus.WARNING) {
        return 'gold';
    }
    return 'blue';
}

const statusIconTextMap: { [key in ComplianceCheckStatus]: ClusterStatusObject } = {
    [ComplianceCheckStatus.PASS]: {
        icon: <CheckCircleIcon color="var(--pf-global--success-color--100)" />,
        statusText: 'Pass',
        tooltipText: 'Check was successful',
    },
    [ComplianceCheckStatus.FAIL]: {
        icon: <SecurityIcon color="var(--pf-global--danger-color--100)" />,
        statusText: 'Fail',
        tooltipText: 'Check was unsuccessful',
    },
    [ComplianceCheckStatus.ERROR]: {
        icon: <ExclamationTriangleIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Error',
        tooltipText: 'Check ran successfully, but could not complete',
    },
    [ComplianceCheckStatus.INFO]: {
        icon: <ExclamationCircleIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Info',
        tooltipText:
            'Check was successful and found something not severe enough to be considered an error',
    },
    [ComplianceCheckStatus.MANUAL]: {
        icon: <WrenchIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Manual',
        tooltipText: 'Check cannot automatically assess the status and manual check is required',
    },
    [ComplianceCheckStatus.NOT_APPLICABLE]: {
        icon: <BanIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Not Applicable',
        tooltipText: 'Check did not run as it is not applicable',
    },
    [ComplianceCheckStatus.INCONSISTENT]: {
        icon: <UnknownIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Inconsistent',
        tooltipText: 'Different nodes report different results',
    },
    [ComplianceCheckStatus.UNSET_CHECK_STATUS]: {
        icon: <ResourcesEmptyIcon color="var(--pf-global--disabled-color--100)" />,
        statusText: 'Unset', // TODO: ask about this status
        tooltipText: '',
    },
};

export function getClusterResultsStatusObject(status: ComplianceCheckStatus): ClusterStatusObject {
    return statusIconTextMap[status];
}

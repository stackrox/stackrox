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

// Thresholds for compliance status
const DANGER_THRESHOLD = 50;
const WARNING_THRESHOLD = 75;

type LabelColor = LabelProps['color'];

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
            case 'PASS':
                passCount += statusInfo.count;
                break;
            case 'FAIL':
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
                <WrenchIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Manual',
        tooltipText:
            'Manual check requires additional organizational or technical knowledge that is not automatable',
        color: 'grey',
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

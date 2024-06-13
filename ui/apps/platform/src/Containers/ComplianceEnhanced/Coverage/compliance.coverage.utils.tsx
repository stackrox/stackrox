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

import {
    ComplianceCheckStatus,
    ComplianceCheckStatusCount,
    ComplianceCheckStatusEnum,
} from 'services/ComplianceCommon';

// Thresholds for compliance status
const DANGER_THRESHOLD = 50;
const WARNING_THRESHOLD = 75;

type LabelColor = LabelProps['color'];

type ClusterStatusObject = {
    icon: ReactElement;
    statusText: string;
    tooltipText: string;
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
            case ComplianceCheckStatusEnum.PASS:
                passCount += statusInfo.count;
                break;
            case ComplianceCheckStatusEnum.FAIL:
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
    [ComplianceCheckStatusEnum.PASS]: {
        icon: (
            <Icon>
                <CheckCircleIcon color="var(--pf-v5-global--primary-color--100)" />
            </Icon>
        ),
        statusText: 'Pass',
        tooltipText: 'Check was successful',
        color: 'blue',
    },
    [ComplianceCheckStatusEnum.FAIL]: {
        icon: (
            <Icon>
                <SecurityIcon color="var(--pf-v5-global--danger-color--100)" />
            </Icon>
        ),
        statusText: 'Fail',
        tooltipText: 'Check was unsuccessful',
        color: 'red',
    },
    [ComplianceCheckStatusEnum.ERROR]: {
        icon: (
            <Icon>
                <ExclamationTriangleIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Error',
        tooltipText: 'Check ran successfully, but could not complete',
        color: 'grey',
    },
    [ComplianceCheckStatusEnum.INFO]: {
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
    [ComplianceCheckStatusEnum.MANUAL]: {
        icon: (
            <Icon>
                <WrenchIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Manual',
        tooltipText: 'Check cannot automatically assess the status and manual check is required',
        color: 'grey',
    },
    [ComplianceCheckStatusEnum.NOT_APPLICABLE]: {
        icon: (
            <Icon>
                <BanIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Not Applicable',
        tooltipText: 'Check did not run as it is not applicable',
        color: 'grey',
    },
    [ComplianceCheckStatusEnum.INCONSISTENT]: {
        icon: (
            <Icon>
                <UnknownIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Inconsistent',
        tooltipText: 'Different nodes report different results',
        color: 'grey',
    },
    [ComplianceCheckStatusEnum.UNSET_CHECK_STATUS]: {
        icon: (
            <Icon>
                <ResourcesEmptyIcon color="var(--pf-v5-global--disabled-color--100)" />
            </Icon>
        ),
        statusText: 'Unset', // TODO: ask about this status
        tooltipText: '',
        color: 'grey',
    },
};

export function getClusterResultsStatusObject(status: ComplianceCheckStatus): ClusterStatusObject {
    return statusIconTextMap[status];
}

import { LabelProps } from '@patternfly/react-core';

import {
    ComplianceCheckStatus,
    ComplianceCheckStatusCount,
} from 'services/ComplianceEnhancedService';

// Thresholds for compliance status
const DANGER_THRESHOLD = 50;
const WARNING_THRESHOLD = 75;

type LabelColor = LabelProps['color'];

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

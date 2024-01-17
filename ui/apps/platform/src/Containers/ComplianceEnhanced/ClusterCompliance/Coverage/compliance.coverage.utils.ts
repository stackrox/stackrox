import { ProgressVariant } from '@patternfly/react-core';

import {
    ComplianceCheckStatus,
    ComplianceCheckStatusCount,
} from 'services/ComplianceEnhancedService';

export function getPassAndTotalCount(checkStats: ComplianceCheckStatusCount[]): {
    passCount: number;
    totalCount: number;
} {
    let totalCount = 0;
    let passCount = 0;

    checkStats.forEach((stat) => {
        totalCount += stat.count;
        if (stat.status === ComplianceCheckStatus.PASS) {
            passCount += stat.count;
        }
    });

    return { passCount, totalCount };
}

export function calculateCompliancePercentage(passCount: number, totalCount: number): number {
    return totalCount > 0 ? (passCount / totalCount) * 100 : 0;
}

export function getProgressBarVariant(passPercentage: number): ProgressVariant | undefined {
    let progressVariant: ProgressVariant | undefined;

    if (passPercentage < 50) {
        progressVariant = ProgressVariant.danger;
    } else if (passPercentage < 75) {
        progressVariant = ProgressVariant.warning;
    }

    return progressVariant;
}

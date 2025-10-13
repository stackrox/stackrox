import {
    COMPLIANCE_PASS_COLOR,
    CRITICAL_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
    noViolationsColor,
} from 'constants/severityColors';

export function getColor(value: number) {
    if (value === 100) {
        return COMPLIANCE_PASS_COLOR;
    }
    if (value >= 70) {
        return MODERATE_MEDIUM_SEVERITY_COLOR;
    }
    if (value >= 50) {
        return IMPORTANT_HIGH_SEVERITY_COLOR;
    }
    if (Number.isNaN(value)) {
        return noViolationsColor; // skipped
    }
    return CRITICAL_SEVERITY_COLOR;
}

// For example, Passing Standards by Clusters chart.
export const verticalBarColors = [
    'var(--base-700)',
    'var(--primary-700)',
    'var(--secondary-700)',
    'var(--base-400)',
    'var(--primary-400)',
    'var(--secondary-400)',
];

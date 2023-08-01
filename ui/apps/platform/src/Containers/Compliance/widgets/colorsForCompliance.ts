import {
    COMPLIANCE_PASS_COLOR,
    CRITICAL_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
    noViolationsColor,
} from 'constants/visuals/colors';

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

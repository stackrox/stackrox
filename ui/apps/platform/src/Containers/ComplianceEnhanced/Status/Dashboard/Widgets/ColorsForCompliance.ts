import {
    CRITICAL_SEVERITY_COLOR,
    IMPORTANT_HIGH_SEVERITY_COLOR,
    LOW_SEVERITY_COLOR,
    MODERATE_MEDIUM_SEVERITY_COLOR,
} from 'constants/severityColors';

export function getBarColor(value: number) {
    if (value === 100) {
        return LOW_SEVERITY_COLOR;
    }
    if (value >= 70) {
        return MODERATE_MEDIUM_SEVERITY_COLOR;
    }
    if (value >= 50) {
        return IMPORTANT_HIGH_SEVERITY_COLOR;
    }
    return CRITICAL_SEVERITY_COLOR;
}

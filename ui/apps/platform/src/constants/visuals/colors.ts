import { VulnerabilitySeverity } from 'types/cve.proto';
import { PolicySeverity } from 'types/policy.proto';

// For example, vertical bars in Compliance Passing Standards by Clusters chart.
const colors = [
    'var(--base-700)',
    'var(--primary-700)',
    'var(--secondary-700)',
    'var(--base-400)',
    'var(--primary-400)',
    'var(--secondary-400)',
];

export const colorTypes = ['alert', 'caution', 'warning', 'success', 'tertiary', 'primary', 'base'];

export const fileUploadColors = {
    BACKGROUND_COLOR: 'var(--warning-300)', // close to original upload background '#faecd2'
    ICON_COLOR: 'var(--warning-700)', // close to original upload icon '#b39357
};

export const defaultColorType = 'base';

// TODO supersede with policySeverityColors below.
export const severityColors: Record<PolicySeverity, string> = {
    LOW_SEVERITY: 'var(--color-severity-low)',
    MEDIUM_SEVERITY: 'var(--color-severity-medium)',
    HIGH_SEVERITY: 'var(--color-severity-important)',
    CRITICAL_SEVERITY: 'var(--color-severity-critical)',
};

/*
 * Export individual constants for consistency in pseudo-severity use cases like compliance.
 * Vulnerability severity name preceded policy severity name when they differ.
 */
export const LOW_SEVERITY_COLOR = 'var(--pf-global--palette--blue-300)';
export const MODERATE_MEDIUM_SEVERITY_COLOR = 'var(--pf-global--palette--gold-300)';
export const IMPORTANT_HIGH_SEVERITY_COLOR = 'var(--pf-global--palette--orange-200)';
export const CRITICAL_SEVERITY_COLOR = 'var(--pf-global--palette--red-100)';
export const UNKNOWN_SEVERITY_COLOR = 'var(--pf-global--palette--black-400)';

export const COMPLIANCE_PASS_COLOR = LOW_SEVERITY_COLOR; // so long as LOW_SEVERITY_COLOR is blue!
export const COMPLIANCE_FAIL_COLOR = CRITICAL_SEVERITY_COLOR;

export const policySeverityColorMap: Record<PolicySeverity, string> = {
    LOW_SEVERITY: LOW_SEVERITY_COLOR,
    MEDIUM_SEVERITY: MODERATE_MEDIUM_SEVERITY_COLOR,
    HIGH_SEVERITY: IMPORTANT_HIGH_SEVERITY_COLOR,
    CRITICAL_SEVERITY: UNKNOWN_SEVERITY_COLOR,
};

// TODO rename as vulnerabilitySeverityColorMap.
// TODO include Icon in name only if color text below is confirmed.
export const vulnSeverityIconColors: Record<VulnerabilitySeverity, string> = {
    LOW_VULNERABILITY_SEVERITY: LOW_SEVERITY_COLOR,
    MODERATE_VULNERABILITY_SEVERITY: MODERATE_MEDIUM_SEVERITY_COLOR,
    IMPORTANT_VULNERABILITY_SEVERITY: IMPORTANT_HIGH_SEVERITY_COLOR,
    CRITICAL_VULNERABILITY_SEVERITY: CRITICAL_SEVERITY_COLOR,
    UNKNOWN_VULNERABILITY_SEVERITY: UNKNOWN_SEVERITY_COLOR,
};

export const vulnSeverityTextColors: Record<VulnerabilitySeverity, string> = {
    LOW_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--blue-500)',
    MODERATE_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--gold-600)',
    IMPORTANT_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--orange-500)',
    CRITICAL_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--red-200)',
    UNKNOWN_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--black-400)',
};

export default colors;

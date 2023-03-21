import { VulnerabilitySeverity } from 'types/cve.proto';
import { PolicySeverity } from 'types/policy.proto';

const colors = [
    'var(--primary-400)',
    'var(--secondary-400)',
    'var(--tertiary-400)',
    'var(--accent-400)',
    'var(--secondary-500)',
];

export const colorTypes = [
    'alert',
    'caution',
    'warning',
    'success',
    'accent',
    'tertiary',
    'secondary',
    'primary',
    'base',
];

export const fileUploadColors = {
    BACKGROUND_COLOR: 'var(--warning-300)', // close to original upload background '#faecd2'
    ICON_COLOR: 'var(--warning-700)', // close to original upload icon '#b39357
};

export const defaultColorType = 'base';

export const severityColors: Record<PolicySeverity, string> = {
    LOW_SEVERITY: 'var(--color-severity-low)',
    MEDIUM_SEVERITY: 'var(--color-severity-medium)',
    HIGH_SEVERITY: 'var(--color-severity-important)',
    CRITICAL_SEVERITY: 'var(--color-severity-critical)',
};

export const vulnSeverityIconColors: Record<VulnerabilitySeverity, string> = {
    LOW_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--blue-300)',
    MODERATE_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--gold-300)',
    IMPORTANT_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--orange-200)',
    CRITICAL_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--red-100)',
    UNKNOWN_VULNERABILITY_SEVERITY: 'var(--pf-global--palette--black-400)',
};

export default colors;

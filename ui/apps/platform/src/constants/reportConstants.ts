import { VulnerabilitySeverity } from 'types/cve.proto';
import { Fixability } from 'types/report.proto';

type VulnerabilitySeverityLabels = Record<VulnerabilitySeverity, string>;

export const vulnerabilitySeverityLabels: VulnerabilitySeverityLabels = {
    CRITICAL_VULNERABILITY_SEVERITY: 'Critical',
    IMPORTANT_VULNERABILITY_SEVERITY: 'Important',
    MODERATE_VULNERABILITY_SEVERITY: 'Medium',
    LOW_VULNERABILITY_SEVERITY: 'Low',
    UNKNOWN_VULNERABILITY_SEVERITY: 'Unknown',
};

export type FixabilityLabelKey = Exclude<Fixability, 'BOTH' | 'UNSET'>;
type FixabilityLabels = Record<FixabilityLabelKey, string>;

export const fixabilityLabels: FixabilityLabels = {
    FIXABLE: 'Fixable',
    NOT_FIXABLE: 'Unfixable',
};

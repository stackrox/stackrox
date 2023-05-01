import { VulnerabilitySeverity } from 'types/cve.proto';

export const severityRankings: Record<VulnerabilitySeverity, number> = {
    UNKNOWN_VULNERABILITY_SEVERITY: 0,
    LOW_VULNERABILITY_SEVERITY: 1,
    MODERATE_VULNERABILITY_SEVERITY: 2,
    IMPORTANT_VULNERABILITY_SEVERITY: 3,
    CRITICAL_VULNERABILITY_SEVERITY: 4,
};

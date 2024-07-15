import { ObservedCveMode } from 'Containers/Vulnerabilities/types';
import { VulnerabilityState } from 'types/cve.proto';
import { ensureExhaustive } from 'utils/type.utils';

export function getViewStateTitle(
    vulnerabilityState: VulnerabilityState,
    cveViewingMode: ObservedCveMode
): string {
    switch (vulnerabilityState) {
        case 'OBSERVED':
            return cveViewingMode === 'WITH_CVES'
                ? 'Image vulnerabilities'
                : 'Images without vulnerabilities';
        case 'DEFERRED':
            return 'Deferred vulnerabilities';
        case 'FALSE_POSITIVE':
            return 'False-positive vulnerabilities';
        default:
            return ensureExhaustive(vulnerabilityState);
    }
}

export function getViewStateDescription(
    vulnerabilityState: VulnerabilityState,
    cveViewingMode: ObservedCveMode
): string {
    switch (vulnerabilityState) {
        case 'OBSERVED':
            return cveViewingMode === 'WITH_CVES'
                ? 'Images and deployments with observed CVEs'
                : 'Images and deployments without observed CVEs (results might include false negatives due to scanner limitations, such as unsupported operating systems)';
        case 'DEFERRED':
            return 'Observed vulnerabilities that are approved by administrators to be deferred for a period of time or until fixable';
        case 'FALSE_POSITIVE':
            return 'Observed vulnerabilities that are approved by administrators to be marked as false-positives indefinitely';
        default:
            return ensureExhaustive(vulnerabilityState);
    }
}

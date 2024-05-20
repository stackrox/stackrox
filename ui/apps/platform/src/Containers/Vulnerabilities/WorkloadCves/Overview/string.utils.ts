import { ObservedCveMode } from 'Containers/Vulnerabilities/types';
import _ from 'lodash';
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
                ? 'Images and deployments observed with CVEs'
                : 'Images and deployments observed without CVEs (results may be inaccurate due to scanner errors)';
        case 'DEFERRED':
            return 'Observed vulnerabilities that are approved by administrators to be deferred for a period of time or until fixable';
        case 'FALSE_POSITIVE':
            return 'Observed vulnerabilities that are approved by administrators to be marked as false-positives indefinitely';
        default:
            return ensureExhaustive(vulnerabilityState);
    }
}

import {
    VulnerabilityException,
    VulnerabilityExceptionScope,
    isDeferralException,
} from 'services/VulnerabilityExceptionService';

export function getImageScopeSearchValue({ imageScope }: VulnerabilityExceptionScope): string {
    const { registry, remote, tag } = imageScope;
    if (registry === '.*' && remote === '.*' && tag === '.*') {
        return '';
    }
    if (tag === '.*') {
        return `${registry}/${remote}`;
    }
    return `${registry}/${remote}:${tag}`;
}

export function getVulnerabilityState(exception: VulnerabilityException) {
    if (exception.status === 'APPROVED' || exception.status === 'APPROVED_PENDING_UPDATE') {
        return isDeferralException(exception) ? 'DEFERRED' : 'FALSE_POSITIVE';
    }
    return 'OBSERVED';
}

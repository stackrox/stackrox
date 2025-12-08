import { isDeferralException } from 'services/VulnerabilityExceptionService';
import type {
    VulnerabilityException,
    VulnerabilityExceptionScope,
} from 'services/VulnerabilityExceptionService';

export function getImageScopeSearchValue({ imageScope }: VulnerabilityExceptionScope): string {
    const { registry, remote, tag } = imageScope;
    if (registry === '.*' && remote === '.*' && tag === '.*') {
        return '';
    }
    if (tag === '.*') {
        return `${registry}/${remote}`;
    }
    if (tag === '') {
        // TODO (dv 2024-01-17)
        //      If tag is an empty string then the image is referenced directly by hash. This is currently not supported
        //      by deferrals, and will be treated the same as passing a wildcard '.*' as tag. Leaving the tag empty and providing
        //      an 'Image' search query ending with `:` will result in an empty response, so we strip that character off here.
        //
        //      See ROX-20929 to track the BE implementation of hash support and update this code accordingly.
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

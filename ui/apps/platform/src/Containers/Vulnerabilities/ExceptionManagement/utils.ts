import { VulnerabilityExceptionScope } from 'services/VulnerabilityExceptionService';

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

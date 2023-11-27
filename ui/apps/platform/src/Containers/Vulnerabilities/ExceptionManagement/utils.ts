import { VulnerabilityExceptionScope } from 'services/VulnerabilityExceptionService';

export function getImageScope(scope: VulnerabilityExceptionScope): string {
    if (
        scope.imageScope.registry === '.*' &&
        scope.imageScope.remote === '.*' &&
        scope.imageScope.tag === '.*'
    ) {
        return '';
    }
    if (scope.imageScope.tag === '.*') {
        return `${scope.imageScope.registry}/${scope.imageScope.remote}`;
    }
    return `${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`;
}

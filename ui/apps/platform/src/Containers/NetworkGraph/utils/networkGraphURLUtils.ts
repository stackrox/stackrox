import { networkBasePath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';
import { SearchFilter } from 'types/search';

type GetURLLinkToDeploymentParams = {
    cluster: string;
    namespace: string;
    deploymentId: string;
};

export function getURLLinkToDeployment({
    cluster,
    namespace,
    deploymentId,
}: GetURLLinkToDeploymentParams) {
    const queryString = getQueryString({
        s: {
            Cluster: cluster,
            Namespace: namespace,
        },
    });
    const networkGraphLink = `${networkBasePath}/deployment/${deploymentId}${queryString}`;
    return networkGraphLink;
}

export function getPropertiesForAnalytics(searchFilter: SearchFilter) {
    const cluster = searchFilter?.Cluster?.toString() || 'unknown cluster';
    const namespaces = searchFilter?.Namespace?.toString() || '';
    const deployments = searchFilter?.Deployment?.toString() || '';

    return {
        cluster,
        namespaces,
        deployments,
    };
}

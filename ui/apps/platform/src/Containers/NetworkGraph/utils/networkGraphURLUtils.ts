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
    const cluster = searchFilter?.Cluster?.toString() ? 1 : 0;
    const namespaces = searchFilter?.Namespace?.length || 0;
    const deployments = searchFilter?.Deployment?.length || 0;

    return {
        cluster,
        namespaces,
        deployments,
    };
}

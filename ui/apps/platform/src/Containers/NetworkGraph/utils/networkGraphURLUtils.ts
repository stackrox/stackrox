import { networkBasePathPF } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';

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
    const networkGraphLink = `${networkBasePathPF}/deployment/${deploymentId}${queryString}`;
    return networkGraphLink;
}

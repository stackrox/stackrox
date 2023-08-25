import { L4Protocol } from 'types/networkFlow.proto';
import { ProcessSignal } from 'types/processIndicator.proto';
import axios from './instance';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

export const listeningEndpointsBaseUrl = '/v1/listening_endpoints';

/*
A destroyed ProcessListeningOnPort will sometimes be returned by the API before it can be pruned from the database. In these
cases, the following fields will be empty as a JOIN to the table that contains the data is not possible:

podUid
processSignal.id
processSignal.containerId
processSignal.time
processSignal.pid
processSignal.uid
processSignal.gid
processSignal.lineage
processSignal.scraped
processSignal.lineageInfo
clusterId
namespace
containerStartTime
imageId

In most cases this will result in an empty string, but fields that represent time as a string will return `null`:

processSignal.time
containerStartTime
*/
export type ProcessListeningOnPort = {
    endpoint: {
        port: number;
        protocol: L4Protocol;
    };
    clusterId: string;
    namespace: string;
    deploymentId: string;
    imageId: string;
    containerName: string;
    podId: string;
    podUid: string;
    signal: ProcessSignal;
    containerStartTime: string | null;
};

/**
 * Get all listening endpoints for a deployment
 */
export function getListeningEndpointsForDeployment(
    deploymentId: string
): CancellableRequest<ProcessListeningOnPort[]> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{
                listeningEndpoints: ProcessListeningOnPort[];
            }>(`${listeningEndpointsBaseUrl}/deployment/${deploymentId}`, { signal })
            .then((response) => response.data.listeningEndpoints)
    );
}

import { useEffect, useState } from 'react';

import { fetchNetworkBaselineStatuses } from 'services/NetworkService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { EdgeState } from '../components/EdgeStateSelect';
import { BaselineStatus, BaselineStatusType, Flow } from '../types/flow.type';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import {
    getNetworkFlows,
    getUniqueIdFromFlow,
    getUniqueIdFromPeer,
    transformFlowsToPeers,
} from '../utils/flowUtils';

type Result = {
    isLoading: boolean;
    data: { networkFlows: Flow[] };
    error: string;
};

type FetchNetworkFlowsParams = {
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    deploymentId: string;
    edgeState: EdgeState;
};

type FetchNetworkFlowsResult = {
    refetchFlows: () => void;
} & Result;

const defaultResultState = {
    data: { networkFlows: [] },
    error: '',
    isLoading: true,
};

function useFetchNetworkFlows({
    nodes,
    edges,
    deploymentId,
    edgeState,
}: FetchNetworkFlowsParams): FetchNetworkFlowsResult {
    const [result, setResult] = useState<Result>(defaultResultState);

    function fetchFlows() {
        setResult({ data: { networkFlows: [] }, isLoading: true, error: '' });
        const flows = getNetworkFlows(nodes, edges, deploymentId);
        const peers = transformFlowsToPeers(flows);
        fetchNetworkBaselineStatuses({ deploymentId, peers })
            .then((response: { statuses: BaselineStatus[] }) => {
                const statusMap = response.statuses.reduce(
                    (acc, curr) => {
                        const id = getUniqueIdFromPeer(curr.peer);
                        acc[id] = curr.status;
                        return acc;
                    },
                    {} as Record<string, BaselineStatusType>
                );
                const modifiedFlows = flows.map((flow) => {
                    const id = getUniqueIdFromFlow(flow);
                    const modifiedFlow: Flow = {
                        ...flow,
                        isAnomalous: statusMap[id] === 'ANOMALOUS',
                    };
                    return modifiedFlow;
                });
                setResult({
                    isLoading: false,
                    data: { networkFlows: modifiedFlows },
                    error: '',
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    isLoading: false,
                    data: { networkFlows: [] },
                    error: errorMessage,
                });
            });
    }

    useEffect(() => {
        fetchFlows();
        return () => setResult(defaultResultState);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deploymentId, edgeState]);

    return { ...result, refetchFlows: fetchFlows };
}

export default useFetchNetworkFlows;

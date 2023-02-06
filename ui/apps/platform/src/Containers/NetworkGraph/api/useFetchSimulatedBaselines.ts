import { useEffect, useState } from 'react';

import { fetchBaselineComparison } from 'services/NetworkService';
import { GroupedDiffFlows, DiffFlowsResponse } from 'types/networkPolicyService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { Flow } from '../types/flow.type';
import { createFlowsFromGroupedDiffFlows } from '../utils/flowUtils';

type Result = {
    isLoading: boolean;
    data: { simulatedBaselines: Flow[] };
    error: string;
};

type FetchSimulatedBaselinesResult = {
    refetchSimulatedBaselines: () => void;
} & Result;

const defaultResultState = {
    isLoading: true,
    data: { simulatedBaselines: [] },
    error: '',
};

function useFetchSimulatedBaselines(deploymentId): FetchSimulatedBaselinesResult {
    const [result, setResult] = useState<Result>(defaultResultState);

    function fetchSimulatedBaselines() {
        fetchBaselineComparison({ deploymentId })
            .then((response: DiffFlowsResponse) => {
                let simulatedBaselines: Flow[] = [];

                // get added baselines
                const addedBaselines = response.added.reduce((acc, currGroupedDiffFlow) => {
                    const flows = createFlowsFromGroupedDiffFlows(currGroupedDiffFlow, 'ADDED');
                    return [...acc, ...flows];
                }, [] as Flow[]);

                // get removed baselines
                const removedBaselines = response.removed.reduce((acc, currGroupedDiffFlow) => {
                    const flows = createFlowsFromGroupedDiffFlows(currGroupedDiffFlow, 'REMOVED');
                    return [...acc, ...flows];
                }, [] as Flow[]);

                // get reconciled baselines
                const reconciledBaselines = response.reconciled.reduce(
                    (acc, currReconciledDiffFlow) => {
                        const { entity, added, removed, unchanged } = currReconciledDiffFlow;

                        const addedGroupedDiffFlow: GroupedDiffFlows = {
                            entity,
                            properties: added,
                        };
                        const removedGroupedDiffFlow: GroupedDiffFlows = {
                            entity,
                            properties: removed,
                        };
                        const unchangedGroupedDiffFlow: GroupedDiffFlows = {
                            entity,
                            properties: unchanged,
                        };

                        const addedFlows = createFlowsFromGroupedDiffFlows(
                            addedGroupedDiffFlow,
                            'ADDED'
                        );
                        const removedFlows = createFlowsFromGroupedDiffFlows(
                            removedGroupedDiffFlow,
                            'REMOVED'
                        );
                        const unchangedFlows = createFlowsFromGroupedDiffFlows(
                            unchangedGroupedDiffFlow,
                            'UNCHANGED'
                        );

                        return [...acc, ...addedFlows, ...removedFlows, ...unchangedFlows];
                    },
                    [] as Flow[]
                );

                simulatedBaselines = [
                    ...addedBaselines,
                    ...removedBaselines,
                    ...reconciledBaselines,
                ];

                setResult({
                    isLoading: false,
                    data: { simulatedBaselines },
                    error: '',
                });
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({
                    isLoading: false,
                    data: { simulatedBaselines: [] },
                    error: errorMessage,
                });
            });
    }

    useEffect(() => {
        fetchSimulatedBaselines();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [deploymentId]);

    return { ...result, refetchSimulatedBaselines: fetchSimulatedBaselines };
}

export default useFetchSimulatedBaselines;

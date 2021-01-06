import { useState, useCallback } from 'react';
import { useDispatch } from 'react-redux';

import { actions } from 'reducers/network/graph';
import { networkFlowStatus } from 'constants/networkGraph';
import { markNetworkBaselineStatuses } from 'services/NetworkService';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';

export type MarkNetworkBaselines = (networkBaselines: FlattenedNetworkBaseline[]) => void;

function useToggleBaselineStatuses(deploymentId: string): MarkNetworkBaselines {
    const dispatch = useDispatch();
    const [, setResult] = useState({ data: null, error: null, isLoading: false });
    const toggleBaselineStatuses = useCallback(
        (networkBaselines) => {
            setResult((prevState) => ({ ...prevState, isLoading: true }));
            const toggledNetworkBaselines = networkBaselines.map((networkBaseline) => {
                const { status: previousStatus } = networkBaseline;
                return {
                    ...networkBaseline,
                    status:
                        previousStatus === networkFlowStatus.ANOMALOUS
                            ? networkFlowStatus.BASELINE
                            : networkFlowStatus.ANOMALOUS,
                };
            });
            const promise = markNetworkBaselineStatuses({
                deploymentId,
                networkBaselines: toggledNetworkBaselines,
            });
            promise
                .then((response) => {
                    setResult({ data: response.data, isLoading: false, error: null });
                    dispatch(actions.updateNetworkNodes());
                })
                .catch((error) => {
                    setResult({ data: null, isLoading: false, error });
                });
        },
        [deploymentId, dispatch]
    );
    return toggleBaselineStatuses;
}

export default useToggleBaselineStatuses;

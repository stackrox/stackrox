import { useState } from 'react';
import { markNetworkBaselineStatuses } from 'services/NetworkService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { Flow } from '../types/flow.type';

type FlowStatus = 'BASELINE' | 'ANOMALOUS';

type Result = {
    isModifying: boolean;
    error: string;
};

type ModifyBaselineStatuses = {
    modifyBaselineStatuses: (
        flows: Flow[],
        status: FlowStatus,
        onSuccessCallback: () => void
    ) => void;
} & Result;

const defaultResult = {
    isModifying: false,
    error: '',
};

function transformFlowsToPeers(flows: Flow[], status: FlowStatus) {
    return flows.map((flow) => {
        const { entityId, type, entity, namespace, direction, port, protocol } = flow;
        let backendType: string = type;
        if (type === 'CIDR_BLOCK') {
            backendType = 'EXTERNAL_SOURCE';
        } else if (type === 'EXTERNAL_ENTITIES') {
            backendType = 'INTERNET';
        }
        const peer = {
            peer: {
                entity: {
                    id: entityId,
                    name: entity,
                    namespace,
                    type: backendType,
                },
                ingress: direction === 'Ingress',
                port,
                protocol,
            },
            status,
        };
        return peer;
    });
}

function useModifyBaselineStatuses(deploymentId): ModifyBaselineStatuses {
    const [result, setResult] = useState<Result>(defaultResult);

    function modifyBaselineStatuses(
        flows: Flow[],
        status: FlowStatus,
        onSuccessCallback: () => void
    ) {
        setResult({ isModifying: true, error: '' });
        const peers = transformFlowsToPeers(flows, status);
        markNetworkBaselineStatuses({
            deploymentId,
            networkBaselines: peers,
        })
            .then(() => {
                setResult({ isModifying: false, error: '' });
                onSuccessCallback();
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                const errorMessage =
                    message || 'An unknown error occurred while getting the list of clusters';

                setResult({ isModifying: false, error: errorMessage });
            });
    }

    return {
        ...result,
        modifyBaselineStatuses,
    };
}

export default useModifyBaselineStatuses;

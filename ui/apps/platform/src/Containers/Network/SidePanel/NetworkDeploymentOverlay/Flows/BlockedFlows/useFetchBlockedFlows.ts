import { useEffect, useState } from 'react';

import { FlattenedBlockedFlows } from 'Containers/Network/networkTypes';

type FetchBlockedFlowsResult = {
    isLoading: boolean;
    data: { blockedFlows: FlattenedBlockedFlows[] };
    error: string | null;
};

const defaultResultState = {
    data: { blockedFlows: [] },
    error: null,
    isLoading: true,
};

function useFetchBlockedFlows({
    selectedDeployment,
    deploymentId,
    filterState,
}): FetchBlockedFlowsResult {
    const [result, setResult] = useState<FetchBlockedFlowsResult>(defaultResultState);

    useEffect(() => {
        // TODO: Fill this section in
        setResult({ ...defaultResultState, isLoading: false });

        // TODO: Possibly use another value other than selectedDeployment to ensure this logic
        // is executed again. See following comment: https://github.com/stackrox/stackrox/pull/7254#discussion_r555252326
    }, [selectedDeployment, deploymentId, filterState]);

    return result;
}

export default useFetchBlockedFlows;

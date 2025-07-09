import React from 'react';
import pluralize from 'pluralize';
import { Button } from '@patternfly/react-core';

import useInterval from 'hooks/useInterval';
import { fetchNodeUpdates } from 'services/NetworkService';

type NodeUpdateSectionProps = {
    isLoading: boolean;
    lastUpdatedTime: string;
    namespacesFromUrl: string[];
    nodeUpdatesCount: number;
    selectedClusterId: string;
    setCurrentEpochCount: (number) => void;
    updateNetworkNodes: () => void;
};

const NodeUpdateSection = ({
    isLoading,
    lastUpdatedTime,
    namespacesFromUrl,
    nodeUpdatesCount,
    selectedClusterId,
    setCurrentEpochCount,
    updateNetworkNodes,
}: NodeUpdateSectionProps) => {
    // Update the poll epoch after 30 seconds to update the node count for a cluster.
    useInterval(() => {
        if (selectedClusterId && namespacesFromUrl.length > 0) {
            fetchNodeUpdates(selectedClusterId)
                .then((result) => {
                    setCurrentEpochCount(result?.response?.epoch ?? 0);
                })
                .catch(() => {
                    // failure to update the node count is not critical
                });
        }
    }, 30000);

    if (!isLoading && nodeUpdatesCount > 0) {
        return (
            <Button
                variant="secondary"
                onClick={updateNetworkNodes}
                aria-label="Click to refresh the graph"
            >
                {nodeUpdatesCount} {pluralize('update', nodeUpdatesCount)} available
            </Button>
        );
    }

    return <>Last updated {lastUpdatedTime ? `at ${lastUpdatedTime}` : 'never'}</>;
};

export default NodeUpdateSection;

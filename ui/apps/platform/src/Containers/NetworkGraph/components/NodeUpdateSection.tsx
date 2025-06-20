import React from 'react';
import pluralize from 'pluralize';
import { Button } from '@patternfly/react-core';

type NodeUpdateSectionProps = {
    isLoading: boolean;
    lastUpdatedTime: string;
    nodeUpdatesCount: number;
    updateNetworkNodes: () => void;
};

const NodeUpdateSection = ({
    isLoading,
    lastUpdatedTime,
    nodeUpdatesCount,
    updateNetworkNodes,
}: NodeUpdateSectionProps) => {
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

    return <em>Last updated {lastUpdatedTime ? `at ${lastUpdatedTime}` : 'never'}</em>;
};

export default NodeUpdateSection;

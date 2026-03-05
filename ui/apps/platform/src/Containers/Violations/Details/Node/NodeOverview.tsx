import type { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { AlertNode } from 'types/alert.proto';

export type NodeOverviewProps = {
    alertNode: AlertNode;
};

/**
 * Displays an overview of the node associated with a node-type violation,
 * including cluster and node identification fields.
 */
function NodeOverview({ alertNode }: NodeOverviewProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem term="Node name" desc={alertNode.name} />
            <DescriptionListItem term="Node ID" desc={alertNode.id} />
            <DescriptionListItem term="Cluster name" desc={alertNode.clusterName} />
            <DescriptionListItem term="Cluster ID" desc={alertNode.clusterId} />
        </DescriptionList>
    );
}

export default NodeOverview;

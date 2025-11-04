import React from 'react';
import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import ClustersTablePanel from './ClustersTablePanel';
import ClusterPage from './ClusterPage';

function ClustersPage(): ReactElement {
    const { clusterId } = useParams() as { clusterId: string }; // see routePaths for parameter

    if (clusterId) {
        return <ClusterPage clusterId={clusterId} />;
    }

    return <ClustersTablePanel selectedClusterId={clusterId} />;
}

export default ClustersPage;

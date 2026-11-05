import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import NotFoundPage from 'Components/NotFoundPage';

import ClustersTablePanel from './ClustersTablePanel';
import ClusterPage from './ClusterPage';

function ClustersPage(): ReactElement {
    const { clusterId } = useParams() as { clusterId: string }; // see routePaths for parameter

    if (clusterId) {
        // Deleted routes
        if (
            clusterId === 'new' ||
            clusterId.startsWith('init-bundles') ||
            clusterId.startsWith('secure-a-cluster')
        ) {
            return <NotFoundPage />;
        }
        return <ClusterPage clusterId={clusterId} />;
    }

    return <ClustersTablePanel selectedClusterId={clusterId} />;
}

export default ClustersPage;

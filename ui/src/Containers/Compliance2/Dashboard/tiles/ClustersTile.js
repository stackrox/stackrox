import React from 'react';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

import { CLUSTERS_QUERY } from 'queries';
import { resourceLabels } from 'messages/common';

const ClustersTile = () => (
    <Query query={CLUSTERS_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading && data.clusters && Array.isArray(data.clusters)) {
                value = data.clusters.length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.CLUSTER}
                    to="/main/compliance2/clusters"
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default ClustersTile;

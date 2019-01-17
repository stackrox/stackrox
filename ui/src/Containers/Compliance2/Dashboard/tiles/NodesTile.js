import React from 'react';
import flatten from 'lodash/flatten';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

import NODES_QUERY from 'queries/node';
import { resourceLabels } from 'messages/common';

const NodesTile = () => (
    <Query query={NODES_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading && data.clusters && Array.isArray(data.clusters)) {
                value = flatten(data.clusters.map(cluster => cluster.nodes.map(node => node.id)))
                    .length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.NODE}
                    to="/main/compliance2/nodes"
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default NodesTile;

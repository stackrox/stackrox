import React from 'react';
import { NODES_QUERY } from 'queries/node';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';
import flatten from 'lodash/flatten';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
    entityType: entityTypes.NODE
});

const NodesTile = () => (
    <Query query={NODES_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading && data.results && Array.isArray(data.results)) {
                value = flatten(data.results.map(cluster => cluster.nodes.map(node => node.id)))
                    .length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.NODE}
                    to={link.url}
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default NodesTile;

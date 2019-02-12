import React from 'react';
import { CLUSTERS_QUERY } from 'queries/cluster';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
    entityType: entityTypes.CLUSTER
});

const ClustersTile = () => (
    <Query query={CLUSTERS_QUERY} action="list">
        {({ loading, data }) => {
            let value = 0;
            if (!loading && data.results && Array.isArray(data.results)) {
                value = data.results.length;
            }
            return (
                <TileLink
                    value={value}
                    caption={resourceLabels.CLUSTER}
                    to={link.url}
                    loading={loading}
                />
            );
        }}
    </Query>
);

export default ClustersTile;

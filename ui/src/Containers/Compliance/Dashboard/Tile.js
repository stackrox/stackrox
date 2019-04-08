import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTERS_QUERY } from 'queries/cluster';
import { NODES_QUERY } from 'queries/node';
import { ALL_NAMESPACES } from 'queries/namespace';
import { DEPLOYMENTS_QUERY } from 'queries/deployment';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';

import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';

function getQuery(entityType) {
    switch (entityType) {
        case entityTypes.CLUSTER:
            return CLUSTERS_QUERY;
        case entityTypes.NODE:
            return NODES_QUERY;
        case entityTypes.NAMESPACE:
            return ALL_NAMESPACES;
        case entityTypes.DEPLOYMENT:
            return DEPLOYMENTS_QUERY;
        default:
            throw new Error(`Search for ${entityType} not yet implemented`);
    }
}

const DashboardTile = ({ entityType }) => {
    const QUERY = getQuery(entityType);
    const link = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
        entityType
    });
    return (
        <Query query={QUERY} action="list">
            {({ loading, data }) => {
                let value = 0;
                if (!loading && data.results && Array.isArray(data.results)) {
                    value = data.results.length;
                }
                return (
                    <TileLink
                        value={value}
                        caption={resourceLabels[entityType]}
                        to={link.url}
                        loading={loading}
                    />
                );
            }}
        </Query>
    );
};

DashboardTile.propTypes = {
    entityType: PropTypes.string.isRequired
};

export default DashboardTile;

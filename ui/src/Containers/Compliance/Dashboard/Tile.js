import React from 'react';
import PropTypes from 'prop-types';
import { CLUSTERS_QUERY } from 'queries/cluster';
import { NODES_QUERY } from 'queries/node';
import { ALL_NAMESPACES } from 'queries/namespace';
import { DEPLOYMENTS_QUERY } from 'queries/deployment';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import Query from 'Components/ThrowingQuery';
import TileLink from 'Components/TileLink';
import ReactRouterPropTypes from 'react-router-prop-types';

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

const processNumValue = (data, entityType) => {
    let value = 0;
    if (!data || !data.results || !Array.isArray(data.results)) return value;
    if (entityType === entityTypes.CONTROL) {
        const set = new Set();
        data.results.forEach(cluster => {
            cluster.complianceResults.forEach(result => {
                set.add(result.control.id);
            });
        });
        value = set.size;
    } else if (entityType === entityTypes.NODE) {
        value = data.results.reduce((acc, curr) => acc + curr.nodes.length, 0);
    } else {
        value = data.results.length;
    }
    return value;
};

const DashboardTile = ({ match, location, entityType }) => {
    const QUERY = getQuery(entityType);
    const url = URLService.getURL(match, location)
        .base(entityType)
        .url();

    return (
        <Query query={QUERY} action="list">
            {({ loading, data }) => {
                const value = processNumValue(data, entityType);
                return (
                    <TileLink
                        value={value}
                        caption={resourceLabels[entityType]}
                        to={url}
                        loading={loading}
                    />
                );
            }}
        </Query>
    );
};

DashboardTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired
};

export default withRouter(DashboardTile);

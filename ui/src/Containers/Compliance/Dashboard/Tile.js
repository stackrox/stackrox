import React from 'react';
import PropTypes from 'prop-types';
import URLService from 'modules/URLService';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import Query from 'Components/CacheFirstQuery';
import EntityTileLink from 'Components/EntityTileLink';
import ReactRouterPropTypes from 'react-router-prop-types';
import gql from 'graphql-tag';

const CLUSTERS_COUNT = gql`
    query clustersCount {
        results: complianceClusterCount
    }
`;

const NODES_COUNT = gql`
    query nodesCount {
        results: complianceNodeCount
    }
`;

const NAMESPACE_COUNT = gql`
    query namespacesCount {
        results: complianceNamespaceCount
    }
`;

const DEPLOYMENTS_COUNT = gql`
    query deploymentsCount {
        results: complianceDeploymentCount
    }
`;

function getQuery(entityType) {
    switch (entityType) {
        case entityTypes.CLUSTER:
            return CLUSTERS_COUNT;
        case entityTypes.NODE:
            return NODES_COUNT;
        case entityTypes.NAMESPACE:
            return NAMESPACE_COUNT;
        case entityTypes.DEPLOYMENT:
            return DEPLOYMENTS_COUNT;
        default:
            throw new Error(`Search for ${entityType} not yet implemented`);
    }
}

const processNumValue = (data, entityType) => {
    let value = 0;
    if (!data || !data.results) return value;
    if (typeof data.results === 'number') return data.results;
    if (!Array.isArray(data.results)) return value;

    if (entityType === entityTypes.CONTROL) {
        const set = new Set();
        data.results.forEach(cluster => {
            cluster.complianceResults.forEach(result => {
                set.add(result.control.id);
            });
        });
        value = set.size;
    } else {
        value = data.results.length;
    }
    return value;
};

const DashboardTile = ({ match, location, entityType, position }) => {
    const QUERY = getQuery(entityType);
    const url = URLService.getURL(match, location)
        .base(entityType)
        .url();

    return (
        <Query query={QUERY} action="list">
            {({ loading, data }) => {
                const value = processNumValue(data, entityType);
                return (
                    <EntityTileLink
                        count={value}
                        entityType={entityType}
                        url={url}
                        loading={loading}
                        position={position}
                    />
                );
            }}
        </Query>
    );
};

DashboardTile.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    position: PropTypes.string
};

DashboardTile.defaultProps = {
    position: null
};

export default withRouter(DashboardTile);

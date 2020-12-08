import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';

import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'utils/queryService';

import EntityTileLink from 'Components/EntityTileLink';

const NODES_COUNT_QUERY = gql`
    query getNodes($query: String) {
        nodeCount(query: $query)
    }
`;

const getURL = (workflowState) => {
    const url = workflowState.clear().pushList(entityTypes.NODE).toUrl();
    return url;
};

const NodesCountTile = () => {
    const { loading, data = {} } = useQuery(NODES_COUNT_QUERY, {
        variables: {
            query: queryService.objectToWhereClause({}),
        },
    });

    const { nodeCount = 0 } = data;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState);

    return (
        <EntityTileLink
            count={nodeCount}
            entityType={entityTypes.NODE}
            position="middle"
            loading={loading}
            url={url}
            short
        />
    );
};

export default NodesCountTile;

import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';

import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'utils/queryService';

import EntityTileLink from 'Components/EntityTileLink';

const IMAGES_COUNT_QUERY = gql`
    query getImages($query: String) {
        imageCount(query: $query)
    }
`;

const getURL = (workflowState) => {
    const url = workflowState.clear().pushList(entityTypes.IMAGE).toUrl();
    return url;
};

const ImagesCountTile = () => {
    const { loading, data = {} } = useQuery(IMAGES_COUNT_QUERY, {
        variables: {
            query: queryService.objectToWhereClause({}),
        },
    });

    const { imageCount = 0 } = data;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState);

    return (
        <EntityTileLink
            count={imageCount}
            entityType={entityTypes.IMAGE}
            position="middle"
            loading={loading}
            url={url}
            short
        />
    );
};

export default ImagesCountTile;

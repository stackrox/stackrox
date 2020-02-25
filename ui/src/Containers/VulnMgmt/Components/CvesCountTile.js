import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';

import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'modules/queryService';

import EntityTileLink from 'Components/EntityTileLink';

const CVES_COUNT_QUERY = gql`
    query cvesCount($query: String) {
        vulnerabilityCount
        fixableCveCount: vulnerabilityCount(query: $query)
    }
`;

const getURL = workflowState => {
    const url = workflowState
        .clear()
        .pushList(entityTypes.CVE)
        .toUrl();
    return url;
};

const CvesCountTile = () => {
    const { loading, data = {} } = useQuery(CVES_COUNT_QUERY, {
        variables: {
            query: queryService.objectToWhereClause({
                'Fixed By': 'r/.*'
            })
        }
    });

    const { vulnerabilityCount = 0, fixableCveCount = 0 } = data;

    const fixableCveCountText = `(${fixableCveCount} fixable)`;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState);

    return (
        <EntityTileLink
            count={vulnerabilityCount}
            entityType={entityTypes.CVE}
            position="middle"
            subText={fixableCveCountText}
            loading={loading}
            isError={!!fixableCveCount}
            url={url}
            short
        />
    );
};

export default CvesCountTile;

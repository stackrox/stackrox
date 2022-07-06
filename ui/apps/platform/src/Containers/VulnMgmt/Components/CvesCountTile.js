import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { gql, useQuery } from '@apollo/client';

import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'utils/queryService';

import EntityTileLink from 'Components/EntityTileLink';

const cveCountQueriesMap = {
    [entityTypes.CVE]: gql`
        query cvesCount($query: String) {
            vulnerabilityCount
            fixableCveCount: vulnerabilityCount(query: $query)
        }
    `,
    [entityTypes.IMAGE_CVE]: gql`
        query imageCvesCount($query: String) {
            vulnerabilityCount: imageVulnerabilityCount
            fixableCveCount: imageVulnerabilityCount(query: $query)
        }
    `,
    [entityTypes.NODE_CVE]: gql`
        query nodeCvesCount($query: String) {
            vulnerabilityCount: nodeVulnerabilityCount
            fixableCveCount: nodeVulnerabilityCount(query: $query)
        }
    `,
    [entityTypes.CLUSTER_CVE]: gql`
        query clusterCvesCount($query: String) {
            vulnerabilityCount: clusterVulnerabilityCount
            fixableCveCount: clusterVulnerabilityCount(query: $query)
        }
    `,
};

const getURL = (workflowState, entityType) => {
    const url = workflowState.clear().pushList(entityType).toUrl();
    return url;
};

const CvesCountTile = ({ entityType }) => {
    const countsQuery = cveCountQueriesMap[entityType];

    const { loading, data = {} } = useQuery(countsQuery, {
        variables: {
            query: queryService.objectToWhereClause({
                Fixable: true,
            }),
        },
    });

    const { vulnerabilityCount = 0, fixableCveCount = 0 } = data;

    const fixableCveCountText = `(${fixableCveCount} fixable)`;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState, entityType);

    return (
        <EntityTileLink
            count={vulnerabilityCount}
            entityType={entityType}
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

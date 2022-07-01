import React, { useContext } from 'react';
import { gql, useQuery } from '@apollo/client';

import Menu from 'Components/Menu';
import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import queryService from 'utils/queryService';

function getURL(workflowState, entityType) {
    const url = workflowState.clear().pushList(entityType).toUrl();
    return url;
}

const cveCountQueriesMap = {
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

const errorClasses = 'bg-alert-200 hover:bg-alert-300 border-alert-400';

const CvesMenu = () => {
    const workflowState = useContext(workflowStateContext);

    const imageVulnCountsQuery = cveCountQueriesMap[entityTypes.IMAGE_CVE];
    const nodeVulnCountsQuery = cveCountQueriesMap[entityTypes.NODE_CVE];
    const clusterVulnCountsQuery = cveCountQueriesMap[entityTypes.CLUSTER_CVE];

    const { loading: imageVulnLoading, data: imageVulnData = {} } = useQuery(imageVulnCountsQuery, {
        variables: {
            query: queryService.objectToWhereClause({
                Fixable: true,
            }),
        },
    });

    const { loading: nodeVulnLoading, data: nodeVulnData = {} } = useQuery(nodeVulnCountsQuery, {
        variables: {
            query: queryService.objectToWhereClause({
                Fixable: true,
            }),
        },
    });

    const { loading: clusterVulnLoading, data: clusterVulnData = {} } = useQuery(
        clusterVulnCountsQuery,
        {
            variables: {
                query: queryService.objectToWhereClause({
                    Fixable: true,
                }),
            },
        }
    );

    const { vulnerabilityCount: imageVulnCount = 0 } = imageVulnData;
    const { vulnerabilityCount: nodeVulnCount = 0 } = nodeVulnData;
    const { vulnerabilityCount: clusterVulnCount = 0 } = clusterVulnData;

    const options =
        !imageVulnLoading && !nodeVulnLoading && !clusterVulnLoading
            ? [
                  {
                      label: `${imageVulnCount} Image CVEs`,
                      link: getURL(workflowState, entityTypes.IMAGE_CVE),
                  },
                  {
                      label: `${nodeVulnCount} Node CVEs`,
                      link: getURL(workflowState, entityTypes.NODE_CVE),
                  },
                  {
                      label: `${clusterVulnCount} Platform CVEs`,
                      link: getURL(workflowState, entityTypes.CLUSTER_CVE),
                  },
              ]
            : [];

    const menuTitle = `${imageVulnCount + nodeVulnCount + clusterVulnCount} CVEs`;

    return (
        <Menu
            buttonClass={`bg-base-100 hover:bg-base-200 border border-base-400 btn-class flex font-condensed h-full text-base-600 ${errorClasses}`}
            buttonText={menuTitle}
            options={options}
            className="h-full min-w-32"
        />
    );
};

export default CvesMenu;

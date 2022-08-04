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

// TODO: fixable counts are not currently used because of space considerations in the UI, but will come back in Consolidated Workflows
const cveCountsQuery = gql`
    query cvesCount($query: String) {
        imageVulnerabilityCount
        fixableImageVulnerabilityCount: imageVulnerabilityCount(query: $query)
        nodeVulnerabilityCount
        fixableNodeVulnerabilityCount: nodeVulnerabilityCount(query: $query)
        clusterVulnerabilityCount
        fixableClusterVulnerabilityCount: clusterVulnerabilityCount(query: $query)
    }
`;

const errorClasses = 'bg-alert-200 hover:bg-alert-300 border-alert-400';

const CvesMenu = () => {
    const workflowState = useContext(workflowStateContext);

    const { loading, data = {} } = useQuery(cveCountsQuery, {
        variables: {
            query: queryService.objectToWhereClause({
                Fixable: true,
            }),
        },
    });

    const options = !loading
        ? [
              {
                  label: `${data.imageVulnerabilityCount} Image CVEs`,
                  link: getURL(workflowState, entityTypes.IMAGE_CVE),
              },
              {
                  label: `${data.nodeVulnerabilityCount} Node CVEs`,
                  link: getURL(workflowState, entityTypes.NODE_CVE),
              },
              {
                  label: `${data.clusterVulnerabilityCount} Platform CVEs`,
                  link: getURL(workflowState, entityTypes.CLUSTER_CVE),
              },
          ]
        : [];

    const totalCveCount =
        data.imageVulnerabilityCount + data.nodeVulnerabilityCount + data.clusterVulnerabilityCount;
    const menuTitle = `${totalCveCount} CVEs`;

    return (
        <Menu
            buttonClass={`bg-base-100 hover:bg-base-200 border border-base-400 btn-class flex font-condensed h-full text-center text-sm text-base-600 ${errorClasses}`}
            buttonText={menuTitle}
            options={options}
            className="h-full min-w-24"
        />
    );
};

export default CvesMenu;

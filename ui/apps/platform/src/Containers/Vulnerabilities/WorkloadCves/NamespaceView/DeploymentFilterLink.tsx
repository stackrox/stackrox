import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { getQueryString } from 'utils/queryStringUtils';

export type DeploymentFilterLinkProps = {
    deploymentCount: number;
    namespaceName: string;
    clusterName: string;
    vulnMgmtBaseUrl: string;
};

function DeploymentFilterLink({ deploymentCount, namespaceName, clusterName, vulnMgmtBaseUrl }) {
    const query = getQueryString({
        vulnerabilityState: 'OBSERVED',
        entityTab: 'Deployment',
        s: {
            Namespace: `^${namespaceName}$`,
            Cluster: `^${clusterName}$`,
        },
    });
    return (
        <Link to={`${vulnMgmtBaseUrl}${query}`}>{pluralize(deploymentCount, 'deployment')}</Link>
    );
}

export default DeploymentFilterLink;

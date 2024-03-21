import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Link } from 'react-router-dom';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';

export type DeploymentFilterLinkProps = {
    deploymentCount: number;
    namespaceName: string;
    clusterName: string;
};

function DeploymentFilterLink({ deploymentCount, namespaceName, clusterName }) {
    const query = getQueryString({
        vulnerabilityState: 'OBSERVED',
        entityTab: 'Deployment',
        s: {
            NAMESPACE: namespaceName,
            CLUSTER: clusterName,
        },
    });
    return (
        <Link to={`${vulnerabilitiesWorkloadCvesPath}${query}`}>
            {pluralize(deploymentCount, 'deployment')}
        </Link>
    );
}

export default DeploymentFilterLink;

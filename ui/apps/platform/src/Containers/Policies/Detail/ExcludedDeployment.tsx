import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { ClusterScopeObject } from 'services/RolesService';
import { PolicyExcludedDeployment } from 'types/policy.proto';

import { getClusterName } from '../policies.utils';

type ExcludedDeploymentProps = {
    clusters: ClusterScopeObject[];
    excludedDeployment: PolicyExcludedDeployment;
};

function ExcludedDeployment({
    clusters,
    excludedDeployment,
}: ExcludedDeploymentProps): ReactElement {
    const { name: deploymentName, scope } = excludedDeployment;
    const { cluster: clusterId, namespace: namespaceName, label } = scope ?? {};

    return (
        <DescriptionList isCompact isHorizontal>
            {clusterId && (
                <DescriptionListItem term="Cluster" desc={getClusterName(clusters, clusterId)} />
            )}
            {namespaceName && <DescriptionListItem term="Namespace" desc={namespaceName} />}
            {deploymentName && <DescriptionListItem term="Deployment" desc={deploymentName} />}
            {label && <DescriptionListItem term="Label" desc={`${label.key}=${label.value}`} />}
        </DescriptionList>
    );
}

export default ExcludedDeployment;

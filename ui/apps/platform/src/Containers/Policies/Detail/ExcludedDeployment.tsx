import type { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { ClusterScopeObject } from 'services/RolesService';
import type { PolicyExcludedDeployment } from 'types/policy.proto';

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
    const {
        cluster: clusterId,
        clusterLabel,
        namespace: namespaceName,
        namespaceLabel,
        label,
    } = scope ?? {};

    return (
        <DescriptionList isCompact isHorizontal horizontalTermWidthModifier={{ default: '16ch' }}>
            {clusterId && (
                <DescriptionListItem term="Cluster" desc={getClusterName(clusters, clusterId)} />
            )}
            {clusterLabel && (
                <DescriptionListItem
                    term="Cluster label"
                    desc={`${clusterLabel.key}=${clusterLabel.value}`}
                />
            )}
            {namespaceName && <DescriptionListItem term="Namespace" desc={namespaceName} />}
            {namespaceLabel && (
                <DescriptionListItem
                    term="Namespace label"
                    desc={`${namespaceLabel.key}=${namespaceLabel.value}`}
                />
            )}
            {deploymentName && <DescriptionListItem term="Deployment" desc={deploymentName} />}
            {label && (
                <DescriptionListItem term="Deployment label" desc={`${label.key}=${label.value}`} />
            )}
        </DescriptionList>
    );
}

export default ExcludedDeployment;

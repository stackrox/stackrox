import React from 'react';
import { Link } from 'react-router-dom';
import { Flex } from '@patternfly/react-core';
import { Cluster } from 'types/cluster.proto';
import HelmIndicator from './HelmIndicator';
import OperatorIndicator from './OperatorIndicator';

type ClusterNameWithTypeIconProps = {
    cluster: Cluster;
};

function ClusterNameWithTypeIcon({ cluster }: ClusterNameWithTypeIconProps) {
    function isHelmManaged(cluster: Cluster) {
        return (
            cluster.managedBy === 'MANAGER_TYPE_HELM_CHART' ||
            (cluster.managedBy === 'MANAGER_TYPE_UNKNOWN' && !!cluster.helmConfig)
        );
    }

    function isOperatorManaged(cluster: Cluster) {
        return cluster.managedBy === 'MANAGER_TYPE_KUBERNETES_OPERATOR';
    }

    return (
        <Flex
            alignItems={{ default: 'alignItemsCenter' }}
            columnGap={{ default: 'columnGapXs' }}
            flexWrap={{ default: 'nowrap' }}
            data-testid="cluster-name"
        >
            <Link to={cluster.id} className="pf-v5-u-text-break-word">
                {cluster.name}
            </Link>
            {isHelmManaged(cluster) && <HelmIndicator />}
            {isOperatorManaged(cluster) && <OperatorIndicator />}
        </Flex>
    );
}

export default ClusterNameWithTypeIcon;

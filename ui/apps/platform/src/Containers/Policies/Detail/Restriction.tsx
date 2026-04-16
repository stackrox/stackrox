import type { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { ClusterScopeObject } from 'services/RolesService';
import type { PolicyScope } from 'types/policy.proto';

import { getClusterName } from '../policies.utils';

type RestrictionProps = {
    clusters: ClusterScopeObject[];
    restriction: PolicyScope;
};

function Restriction({ clusters, restriction }: RestrictionProps): ReactElement {
    const { cluster: clusterId, clusterLabel, namespace, namespaceLabel, label } = restriction;

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
            {namespace && <DescriptionListItem term="Namespace" desc={namespace} />}
            {namespaceLabel && (
                <DescriptionListItem
                    term="Namespace label"
                    desc={`${namespaceLabel.key}=${namespaceLabel.value}`}
                />
            )}
            {label && (
                <DescriptionListItem term="Deployment label" desc={`${label.key}=${label.value}`} />
            )}
        </DescriptionList>
    );
}

export default Restriction;

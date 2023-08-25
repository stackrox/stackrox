import React, { ReactElement } from 'react';
import { DescriptionList } from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { ClusterScopeObject } from 'services/RolesService';
import { PolicyScope } from 'types/policy.proto';

import { getClusterName } from '../policies.utils';

type RestrictionProps = {
    clusters: ClusterScopeObject[];
    restriction: PolicyScope;
};

function Restriction({ clusters, restriction }: RestrictionProps): ReactElement {
    const { cluster: clusterId, namespace, label } = restriction;

    return (
        <DescriptionList isCompact isHorizontal>
            {clusterId && (
                <DescriptionListItem term="Cluster" desc={getClusterName(clusters, clusterId)} />
            )}
            {namespace && <DescriptionListItem term="Namespace" desc={namespace} />}
            {label && <DescriptionListItem term="Label" desc={`${label.key}=${label.value}`} />}
        </DescriptionList>
    );
}

export default Restriction;

import React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import useURLSearch from 'hooks/useURLSearch';
import ClusterSelector from './ClusterSelector';
import NamespaceSelector from './NamespaceSelector';
import getScopeHierarchy from '../utils/getScopeHierarchy';

type NetworkBreadcrumbsProps = {
    clusters: Cluster[];
};

function NetworkBreadcrumbs({ clusters = [] }: NetworkBreadcrumbsProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    const { cluster: selectedClusterName } = getScopeHierarchy(searchFilter);

    const selectedClusterId =
        clusters.find((cluster) => cluster.name === selectedClusterName)?.id || '';

    return (
        <>
            <Breadcrumb>
                <BreadcrumbItem isDropdown>
                    <ClusterSelector
                        clusters={clusters}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
                <BreadcrumbItem isDropdown>
                    <NamespaceSelector
                        selectedClusterId={selectedClusterId}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
            </Breadcrumb>
        </>
    );
}

export default NetworkBreadcrumbs;

import React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';

import { useFetchClusterNamespacesForPermissions } from 'hooks/useFetchClusterNamespacesForPermissions';
import useFetchNamespaceDeployments from 'hooks/useFetchNamespaceDeployments';
import { nonGlobalResourceNamesForNetworkGraph } from 'routePaths';

import { SearchFilter } from 'types/search';
import ClusterSelector, { ClusterSelectorProps } from './ClusterSelector';
import NamespaceSelector from './NamespaceSelector';
import DeploymentSelector from './DeploymentSelector';

import { useSearchFilter } from '../NetworkGraphURLStateContext';

export type NetworkBreadcrumbsProps = {
    clusters: ClusterSelectorProps['clusters'];
    selectedCluster: { name: string; id: string };
    selectedNamespaces: string[];
    selectedDeployments: string[];
    onScopeChange: (newFilter: SearchFilter) => void;
};

function NetworkBreadcrumbs({
    clusters,
    selectedCluster,
    selectedNamespaces,
    selectedDeployments,
    onScopeChange,
}: NetworkBreadcrumbsProps) {
    const { searchFilter, setSearchFilter } = useSearchFilter();

    const { namespaces } = useFetchClusterNamespacesForPermissions(
        nonGlobalResourceNamesForNetworkGraph,
        selectedCluster?.id
    );
    const selectedNamespaceIds = namespaces.reduce<string[]>((acc: string[], namespace) => {
        return selectedNamespaces.includes(namespace.name) ? [...acc, namespace.id] : acc;
    }, []);
    const { deploymentsByNamespace } = useFetchNamespaceDeployments(selectedNamespaceIds);

    const onChange = (newFilter: SearchFilter) => {
        setSearchFilter(newFilter);
        onScopeChange(newFilter);
    };

    return (
        <Breadcrumb>
            <BreadcrumbItem isDropdown>
                <ClusterSelector
                    clusters={clusters}
                    selectedClusterName={selectedCluster?.name ?? ''}
                    searchFilter={searchFilter}
                    setSearchFilter={onChange}
                />
            </BreadcrumbItem>
            <BreadcrumbItem isDropdown>
                <NamespaceSelector
                    namespaces={namespaces}
                    selectedNamespaces={selectedNamespaces}
                    selectedDeployments={selectedDeployments}
                    deploymentsByNamespace={deploymentsByNamespace}
                    searchFilter={searchFilter}
                    setSearchFilter={onChange}
                />
            </BreadcrumbItem>
            <BreadcrumbItem isDropdown>
                <DeploymentSelector
                    deploymentsByNamespace={deploymentsByNamespace}
                    selectedDeployments={selectedDeployments}
                    searchFilter={searchFilter}
                    setSearchFilter={onChange}
                />
            </BreadcrumbItem>
        </Breadcrumb>
    );
}

export default NetworkBreadcrumbs;

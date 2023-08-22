import React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { useFetchClusterNamespacesForPermissions } from 'hooks/useFetchClusterNamespacesForPermissions';
import useFetchNamespaceDeployments from 'hooks/useFetchNamespaceDeployments';
import { nonGlobalResourceNamesForNetworkGraph } from 'routePaths';

import ClusterSelector, { ClusterSelectorProps } from './ClusterSelector';
import NamespaceSelector from './NamespaceSelector';
import DeploymentSelector from './DeploymentSelector';

export type NetworkBreadcrumbsProps = {
    clusters: ClusterSelectorProps['clusters'];
    selectedCluster: { name: string; id: string };
    selectedNamespaces: string[];
    selectedDeployments: string[];
};

function NetworkBreadcrumbs({
    clusters,
    selectedCluster,
    selectedNamespaces,
    selectedDeployments,
}: NetworkBreadcrumbsProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    const { namespaces } = useFetchClusterNamespacesForPermissions(
        nonGlobalResourceNamesForNetworkGraph,
        selectedCluster?.id
    );
    const selectedNamespaceIds = namespaces.reduce<string[]>((acc: string[], namespace) => {
        return selectedNamespaces.includes(namespace.name) ? [...acc, namespace.id] : acc;
    }, []);
    const { deploymentsByNamespace } = useFetchNamespaceDeployments(selectedNamespaceIds);

    return (
        <>
            <Breadcrumb>
                <BreadcrumbItem isDropdown>
                    <ClusterSelector
                        clusters={clusters}
                        selectedClusterName={selectedCluster?.name ?? ''}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
                <BreadcrumbItem isDropdown>
                    <NamespaceSelector
                        namespaces={namespaces}
                        selectedNamespaces={selectedNamespaces}
                        selectedDeployments={selectedDeployments}
                        deploymentsByNamespace={deploymentsByNamespace}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
                <BreadcrumbItem isDropdown>
                    <DeploymentSelector
                        deploymentsByNamespace={deploymentsByNamespace}
                        selectedDeployments={selectedDeployments}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
            </Breadcrumb>
        </>
    );
}

export default NetworkBreadcrumbs;

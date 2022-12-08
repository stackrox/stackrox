import React from 'react';
import { Breadcrumb, BreadcrumbItem } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import useURLSearch from 'hooks/useURLSearch';
import useFetchClusterNamespaces from 'hooks/useFetchClusterNamespaces';
import useFetchNamespaceDeployments from 'hooks/useFetchNamespaceDeployments';
import ClusterSelector from './ClusterSelector';
import NamespaceSelector from './NamespaceSelector';
// import DeploymentSelector from './DeploymentSelector';

type NetworkBreadcrumbsProps = {
    clusters: Cluster[];
    selectedCluster?: { name?: string; id?: string };
    selectedNamespaces: string[];
};

function NetworkBreadcrumbs({
    clusters = [],
    selectedCluster = {},
    selectedNamespaces = [],
}: NetworkBreadcrumbsProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    const { loading, error, namespaces } = useFetchClusterNamespaces(selectedCluster?.id);
    const selectedNamespaceIds = namespaces.reduce((acc: string[], namespace) => {
        return selectedNamespaces.includes(namespace.metadata.name)
            ? [...acc, namespace.metadata.id]
            : acc;
    }, []);
    const { deploymentsByNamespace } = useFetchNamespaceDeployments(selectedNamespaceIds);
    console.log({ deploymentsByNamespace });

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
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem>
                {/* <BreadcrumbItem isDropdown>
                    <NamespaceSelector
                        selectedNamespaces={selectedDeployments}
                        selectedDeployments={selectedDeployments}
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                    />
                </BreadcrumbItem> */}
            </Breadcrumb>
        </>
    );
}

export default NetworkBreadcrumbs;

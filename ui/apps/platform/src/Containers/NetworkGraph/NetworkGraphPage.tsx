import React from 'react';
import { Flex, FlexItem, PageSection } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import useClusters from './useClusters';
import useNamespaces from './useNamespaces';
import ClusterSelect from './ClusterSelect';
import NamespaceSelect from './NamespaceSelect';

export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
};

function NetworkGraphPage() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const selectedClusterId = (searchFilter.Cluster as string) || '';
    const selectedNamespaces = (searchFilter.Namespace as string[]) || [];

    const { clusters, error: clusterError, isLoading: isLoadingCluster } = useClusters();
    const {
        namespaces,
        error: namespaceError,
        loading: isLoadingNamespace,
    } = useNamespaces(selectedClusterId);

    function updateSelectedClusterId(selection) {
        const newSearchFiliter = { ...searchFilter, Cluster: selection };
        setSearchFilter(newSearchFiliter);
    }

    function updateSelectedNamespaces(selection) {
        const newSearchFiliter = { ...searchFilter, Namespace: selection };
        setSearchFilter(newSearchFiliter);
    }

    return (
        <PageSection variant="light" isFilled id="policies-table-loading">
            <h1>Network Graph</h1>
            <Flex>
                <FlexItem>
                    <ClusterSelect
                        clusters={clusters}
                        selectedClusterId={selectedClusterId}
                        setSelectedClusterId={updateSelectedClusterId}
                        isLoading={isLoadingCluster}
                        error={clusterError}
                    />
                </FlexItem>
                <FlexItem>
                    <NamespaceSelect
                        namespaces={namespaces}
                        selectedNamespaces={selectedNamespaces}
                        setSelectedNamespaces={updateSelectedNamespaces}
                    />
                </FlexItem>
            </Flex>
        </PageSection>
    );
}

export default NetworkGraphPage;

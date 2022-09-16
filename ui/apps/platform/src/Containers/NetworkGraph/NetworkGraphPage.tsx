import React from 'react';
import { Flex, FlexItem, PageSection } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { fetchClustersAsArray } from 'services/ClustersService';
import { Cluster } from 'types/cluster.proto';
import ClusterSelect from './ClusterSelect';
import NamespaceSelect from './NamespaceSelect';

export type Namespace = {
    metadata: {
        id: string;
        name: string;
    };
};

function NetworkGraphPage() {
    const [clusters, setClusters] = React.useState<Cluster[]>([]);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const selectedClusterId = (searchFilter.Cluster as string) || '';
    const selectedNamespaces = (searchFilter.Namespace as string[]) || [];

    React.useEffect(() => {
        fetchClustersAsArray()
            .then((data) => {
                setClusters(data as Cluster[]);
            })
            .catch(() => {
                // TODO
            });
    }, []);

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
                    />
                </FlexItem>
                <FlexItem>
                    <NamespaceSelect
                        selectedClusterId={selectedClusterId}
                        selectedNamespaces={selectedNamespaces}
                        setSelectedNamespaces={updateSelectedNamespaces}
                    />
                </FlexItem>
            </Flex>
        </PageSection>
    );
}

export default NetworkGraphPage;

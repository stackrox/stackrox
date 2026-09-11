import { useMemo } from 'react';
import { Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import omit from 'lodash/omit';

import useURLSearch from 'hooks/useURLSearch';

import { flattenFilterValue } from 'utils/searchUtils';
import NamespaceSelect from './NamespaceSelect';
import ClusterSelect from './ClusterSelect';
import type { SelectionChangeAction } from './ClusterSelect';
import useClustersWithNamespaces from './hooks/useClustersWithNamespaces';

function ScopeBar() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { data, isLoading, error } = useClustersWithNamespaces();
    const selectedClusterData = useMemo(() => {
        return data?.filter(({ name }) => searchFilter.Cluster?.includes(name)) ?? [];
    }, [data, searchFilter.Cluster]);

    function onClusterChange(changeAction: SelectionChangeAction) {
        const { type, value, selection } = changeAction;
        const prevNamespaceIds = flattenFilterValue(searchFilter['Namespace ID'], []);

        // When a cluster is selected, all namespaces belonging to that cluster should be selected.
        if (prevNamespaceIds.length === 0) {
            // If the Namespace filter is currently set to "Select All", don't
            // change it. Adding or removing a cluster should retain this option.
            setSearchFilter({ ...searchFilter, Cluster: selection });
        } else {
            // If the Namespace filter has some value other than "Select All" selected, we need to
            // do a more fine-grained update of the selection. When a new cluster is selected, all
            // namespaces belonging to that cluster should be selected. When a cluster is
            // removed, any selected namespaces that belong to that cluster should be removed.
            const changedCluster = data?.find((cs) => cs.name === value);
            const toggledNamespaceIds =
                changedCluster?.namespaces.map(({ metadata }) => metadata.id) ?? [];
            const selectedNamespaceIds =
                type === 'add'
                    ? [...prevNamespaceIds, ...toggledNamespaceIds]
                    : prevNamespaceIds.filter((ns) => !toggledNamespaceIds.includes(ns));

            setSearchFilter({
                ...searchFilter,
                Cluster: selection,
                'Namespace ID': selectedNamespaceIds,
            });
        }
    }

    function onClusterSelectAll() {
        setSearchFilter(omit(searchFilter, 'Cluster', 'Namespace ID'));
    }

    function onNamespaceChange(namespaceSelection: string[]) {
        setSearchFilter({ ...searchFilter, 'Namespace ID': namespaceSelection });
    }

    function onNamespaceSelectAll() {
        setSearchFilter(omit(searchFilter, 'Namespace ID'));
    }

    return (
        <Toolbar className="pf-v6-u-p-0">
            <ToolbarContent className="pf-v6-u-p-0">
                <ToolbarItem variant="label" className="pf-v6-u-align-self-center">
                    Resources:
                </ToolbarItem>
                <ToolbarItem>
                    <ClusterSelect
                        clusters={data ?? []}
                        clusterSearch={searchFilter.Cluster}
                        isDisabled={isLoading || Boolean(error)}
                        onChange={onClusterChange}
                        onSelectAll={onClusterSelectAll}
                    />
                </ToolbarItem>
                <ToolbarItem>
                    <NamespaceSelect
                        clusters={selectedClusterData}
                        namespaceSearch={searchFilter['Namespace ID']}
                        isDisabled={selectedClusterData.length === 0 || isLoading || Boolean(error)}
                        onChange={onNamespaceChange}
                        onSelectAll={onNamespaceSelectAll}
                    />
                </ToolbarItem>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ScopeBar;

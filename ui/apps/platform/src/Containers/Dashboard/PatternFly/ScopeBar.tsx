import React, { useMemo } from 'react';
import { Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import omit from 'lodash/omit';

import useURLSearch from 'hooks/useURLSearch';

import { flattenFilterValue } from 'utils/searchUtils';
import NamespaceSelect from './NamespaceSelect';
import ClusterSelect, { SelectionChangeAction } from './ClusterSelect';
import { Cluster } from './types';

type NamespacesResponse = {
    clusters: Cluster[];
};

export const namespacesQuery = gql`
    query getAllNamespacesByCluster($query: String) {
        clusters(query: $query) {
            id
            name
            namespaces {
                metadata {
                    id
                    name
                }
            }
        }
    }
`;

function ScopeBar() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { data, loading, error } = useQuery<NamespacesResponse>(namespacesQuery, {
        variables: { query: '' },
    });
    const selectedClusterData = useMemo(() => {
        return data?.clusters.filter(({ name }) => searchFilter.Cluster?.includes(name)) ?? [];
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
            const changedCluster = data?.clusters.find((cs) => cs.name === value);
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
        <Toolbar className="pf-u-p-0">
            <ToolbarContent className="pf-u-p-0">
                <ToolbarItem>
                    <div>Resources:</div>
                </ToolbarItem>
                <ToolbarItem>
                    <ClusterSelect
                        clusters={data?.clusters ?? []}
                        clusterSearch={searchFilter.Cluster}
                        isDisabled={loading || Boolean(error)}
                        onChange={onClusterChange}
                        onSelectAll={onClusterSelectAll}
                    />
                </ToolbarItem>
                <ToolbarItem>
                    <NamespaceSelect
                        clusters={selectedClusterData}
                        namespaceSearch={searchFilter['Namespace ID']}
                        isDisabled={selectedClusterData.length === 0 || loading || Boolean(error)}
                        onChange={onNamespaceChange}
                        onSelectAll={onNamespaceSelectAll}
                    />
                </ToolbarItem>
            </ToolbarContent>
        </Toolbar>
    );
}

export default ScopeBar;

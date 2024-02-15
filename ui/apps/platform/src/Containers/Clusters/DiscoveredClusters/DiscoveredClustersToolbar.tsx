import React, { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import {
    DiscoveredClusterStatus,
    DiscoveredClusterType,
    getDiscoveredClustersFilter,
    isStatus,
    isType,
    replaceSearchFilterNames,
    replaceSearchFilterStatuses,
    replaceSearchFilterTypes,
} from 'services/DiscoveredClusterService';
import { SearchFilter } from 'types/search';

import SearchFilterTypes from './SearchFilterTypes';
import SearchFilterNames from './SearchFilterNames';
import SearchFilterStatuses from './SearchFilterStatuses';
import { getStatusText, getTypeText } from './DiscoveredCluster';

const searchFilterChipDescriptors = [
    { displayName: 'Name', searchFilterName: 'Cluster' },
    {
        displayName: 'Status',
        searchFilterName: 'Cluster Status',
        render: (filter: string) => (isStatus(filter) ? getStatusText(filter) : filter),
    },
    {
        displayName: 'Type',
        searchFilterName: 'Cluster Type',
        render: (filter: string) => (isType(filter) ? getTypeText(filter) : filter),
    },
];

export type DiscoveredClustersToolbarProps = {
    count: number;
    isDisabled: boolean;
    page: number;
    perPage: number;
    setPage: (newPage: number) => void;
    setPerPage: (newPerPage: number) => void;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

function DiscoveredClustersToolbar({
    count,
    isDisabled,
    page,
    perPage,
    setPage,
    setPerPage,
    searchFilter,
    setSearchFilter,
}: DiscoveredClustersToolbarProps): ReactElement {
    function setNamesSelected(names: string[] | undefined) {
        setSearchFilter(replaceSearchFilterNames(searchFilter, names));
    }

    function setStatusesSelected(statuses: DiscoveredClusterStatus[] | undefined) {
        setSearchFilter(replaceSearchFilterStatuses(searchFilter, statuses));
    }

    function setTypesSelected(types: DiscoveredClusterType[] | undefined) {
        setSearchFilter(replaceSearchFilterTypes(searchFilter, types));
    }

    const {
        names: namesSelected,
        types: typesSelected,
        statuses: statusesSelected,
    } = getDiscoveredClustersFilter(searchFilter);

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarItem variant="search-filter">
                    <SearchFilterNames
                        namesSelected={namesSelected}
                        isDisabled={isDisabled}
                        setNamesSelected={setNamesSelected}
                    />
                </ToolbarItem>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterStatuses
                            statusesSelected={statusesSelected}
                            isDisabled={isDisabled}
                            setStatusesSelected={setStatusesSelected}
                        />
                    </ToolbarItem>
                    <ToolbarItem>
                        <SearchFilterTypes
                            typesSelected={typesSelected}
                            isDisabled={isDisabled}
                            setTypesSelected={setTypesSelected}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="button-group" alignment={{ default: 'alignRight' }}>
                    <ToolbarItem variant="pagination">
                        <Pagination
                            isCompact
                            isDisabled={isDisabled}
                            itemCount={count}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPage(1);
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-u-w-100">
                    <SearchFilterChips filterChipGroupDescriptors={searchFilterChipDescriptors} />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default DiscoveredClustersToolbar;

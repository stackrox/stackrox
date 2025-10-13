import React from 'react';
import type { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

// Comment out Names filter for 4.4 MVP because testers expected partial match instead of exact match.

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import {
    getDiscoveredClustersFilter,
    isStatus,
    isType,
    // replaceSearchFilterNames,
    replaceSearchFilterStatuses,
    replaceSearchFilterTypes,
} from 'services/DiscoveredClusterService';
import type {
    DiscoveredClusterStatus,
    DiscoveredClusterType,
} from 'services/DiscoveredClusterService';
import type { SearchFilter } from 'types/search';

import SearchFilterTypes from './SearchFilterTypes';
// import SearchFilterNames from './SearchFilterNames';
import SearchFilterStatuses from './SearchFilterStatuses';
import { getStatusText, getTypeText } from './DiscoveredCluster';

const searchFilterChipDescriptors = [
    // { displayName: 'Name', searchFilterName: 'Cluster' },
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
    // Comment out use of prop for MVP because testers complained about flicker.
    isDisabled: boolean; // eslint-disable-line react/no-unused-prop-types
    page: number;
    perPage: number;
    setPage: (newPage: number) => void;
    setPerPage: (newPerPage: number) => void;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

function DiscoveredClustersToolbar({
    count,
    // Comment out for MVP because testers complained about flicker.
    // isDisabled,
    page,
    perPage,
    setPage,
    setPerPage,
    searchFilter,
    setSearchFilter,
}: DiscoveredClustersToolbarProps): ReactElement {
    /*
    function setNamesSelected(names: string[] | undefined) {
        setSearchFilter(replaceSearchFilterNames(searchFilter, names));
    }
    */

    function setStatusesSelected(statuses: DiscoveredClusterStatus[] | undefined) {
        setSearchFilter(replaceSearchFilterStatuses(searchFilter, statuses));
    }

    function setTypesSelected(types: DiscoveredClusterType[] | undefined) {
        setSearchFilter(replaceSearchFilterTypes(searchFilter, types));
    }

    const {
        // names: namesSelected,
        types: typesSelected,
        statuses: statusesSelected,
    } = getDiscoveredClustersFilter(searchFilter);

    return (
        <Toolbar>
            <ToolbarContent>
                {/*
                <ToolbarItem variant="search-filter">
                    <SearchFilterNames
                        namesSelected={namesSelected}
                        // Comment out for MVP because testers complained about flicker.
                        // isDisabled={isDisabled}
                        setNamesSelected={setNamesSelected}
                    />
                    </ToolbarItem>
                */}
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterStatuses
                            statusesSelected={statusesSelected}
                            // Comment out for MVP because testers complained about flicker.
                            // isDisabled={isDisabled}
                            isDisabled={false}
                            setStatusesSelected={setStatusesSelected}
                        />
                    </ToolbarItem>
                    <ToolbarItem>
                        <SearchFilterTypes
                            typesSelected={typesSelected}
                            // Comment out for MVP because testers complained about flicker.
                            // isDisabled={isDisabled}
                            isDisabled={false}
                            setTypesSelected={setTypesSelected}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup variant="button-group" align={{ default: 'alignRight' }}>
                    <ToolbarItem variant="pagination">
                        <Pagination
                            isCompact
                            // Comment out for MVP because testers complained about flicker.
                            // isDisabled={isDisabled}
                            itemCount={count}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <SearchFilterChips
                        searchFilter={searchFilter}
                        onFilterChange={setSearchFilter}
                        filterChipGroupDescriptors={searchFilterChipDescriptors}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default DiscoveredClustersToolbar;

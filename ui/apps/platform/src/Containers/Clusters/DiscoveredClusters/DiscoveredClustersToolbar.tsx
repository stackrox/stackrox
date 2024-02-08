import React, { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import {
    // DiscoveredClusterStatus,
    DiscoveredClusterType,
    getDiscoveredClustersFilter,
    // replaceSearchFilterName,
    // replaceSearchFilterStatuses,
    replaceSearchFilterTypes,
} from 'services/DiscoveredClusterService';
import { SearchFilter } from 'types/search';

import SearchFilterType from './SearchFilterType';

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
    /*
    function setNameSelected(name: string) {
        setSearchFilter(replaceSearchFilterName(searchFilter, name));
    }

    function setStatusesSelected(types: DiscoveredClusterStatus[] | undefined) {
        setSearchFilter(replaceSearchFilterStatuses(searchFilter, statuses));
    }
    */

    function setTypesSelected(types: DiscoveredClusterType[] | undefined) {
        setSearchFilter(replaceSearchFilterTypes(searchFilter, types));
    }

    const { types: typesSelected } = getDiscoveredClustersFilter(searchFilter);

    return (
        <Toolbar>
            <ToolbarContent>
                {/* SearchFilterName */}
                {/* SearchFilterStatuses */}
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterType
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
            </ToolbarContent>
        </Toolbar>
    );
}

export default DiscoveredClustersToolbar;

import React, { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

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
    function setTypes(/* TODO */) {
        setSearchFilter({ ...searchFilter }); // TODO
    }

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup variant="filter-group">
                    <ToolbarItem>
                        <SearchFilterType
                            types={['GKE']}
                            isDisabled={isDisabled}
                            setTypes={setTypes}
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

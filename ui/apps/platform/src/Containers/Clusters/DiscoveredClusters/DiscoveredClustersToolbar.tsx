import type { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import type { SearchFilter } from 'types/search';

import { searchFilterConfig } from './searchFilterConfig';

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
    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem>
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={(payload) =>
                                setSearchFilter(updateSearchFilter(searchFilter, payload))
                            }
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem>
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={[]}
                            config={searchFilterConfig}
                            onFilterChange={setSearchFilter}
                            searchFilter={searchFilter}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup className="pf-v5-u-w-100">
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
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
            </ToolbarContent>
        </Toolbar>
    );
}

export default DiscoveredClustersToolbar;

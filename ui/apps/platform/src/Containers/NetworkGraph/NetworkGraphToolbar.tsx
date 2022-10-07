/* eslint-disable react/no-array-index-key */
import React from 'react';
import {
    Select,
    SelectOption,
    SelectVariant,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import SearchFilterInput from 'Components/SearchFilterInput';
import { SearchFilter } from 'types/search';

type NetworkGraphToolbarProps = {
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
    searchOptions: string[];
};

const firstOptions = [
    { value: 'Filter 1', disabled: false, isPlaceholder: true },
    { value: 'A', disabled: false },
    { value: 'B', disabled: false },
    { value: 'C', disabled: false },
];

function NetworkGraphToolbar({
    handleChangeSearchFilter,
    searchFilter,
    searchOptions,
}: NetworkGraphToolbarProps): React.ReactElement {
    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarItem
                    variant="search-filter"
                    className="pf-u-flex-grow-1 pf-u-flex-shrink-1"
                >
                    <SearchFilterInput
                        className="w-full theme-light pf-search-shim"
                        handleChangeSearchFilter={handleChangeSearchFilter}
                        placeholder="Filter graph"
                        searchCategory="DEPLOYMENTS"
                        searchFilter={searchFilter ?? {}}
                        searchOptions={searchOptions}
                    />
                </ToolbarItem>
            </ToolbarContent>
        </Toolbar>
    );
}

export default NetworkGraphToolbar;

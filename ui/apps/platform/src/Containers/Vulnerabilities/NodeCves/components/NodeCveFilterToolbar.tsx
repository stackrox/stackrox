import React from 'react';
import noop from 'lodash/noop';
import { Toolbar, ToolbarGroup, ToolbarContent } from '@patternfly/react-core';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { searchValueAsArray } from 'utils/searchUtils';

import CVESeverityDropdown from '../../components/CVESeverityDropdown';
import CVEStatusDropdown from '../../components/CVEStatusDropdown';
import { SearchOption, SearchOptionValue } from '../../searchOptions';
import FilterAutocomplete, {
    FilterAutocompleteSelectProps,
} from '../../components/FilterAutocomplete';

type NodeCveFilterToolbarProps = {
    searchOptions: SearchOption[];
    autocompleteSearchContext?: FilterAutocompleteSelectProps['autocompleteSearchContext'];
    onFilterChange?: (searchFilter: SearchFilter) => void;
};

function NodeCveFilterToolbar({
    searchOptions,
    autocompleteSearchContext,
    onFilterChange = noop,
}: NodeCveFilterToolbarProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();

    function trackAppliedFilter(
        /* eslint-disable @typescript-eslint/no-unused-vars */
        category: SearchOptionValue,
        filter: string
        /* eslint-enable @typescript-eslint/no-unused-vars */
    ) {
        // TODO - track analytics
    }

    function onChangeSearchFilter(newFilter: SearchFilter) {
        setSearchFilter(newFilter);
        onFilterChange(newFilter);
    }

    function onSelect(
        type: Extract<SearchOptionValue, 'SEVERITY' | 'FIXABLE'>,
        checked: boolean,
        selection: string
    ) {
        const selectedSearchFilter = searchValueAsArray(searchFilter[type]);
        onChangeSearchFilter({
            ...searchFilter,
            [type]: checked
                ? [...selectedSearchFilter, selection]
                : selectedSearchFilter.filter((value) => value !== selection),
        });

        if (checked) {
            trackAppliedFilter(type, selection);
        }
    }

    const filterChipGroupDescriptors = [
        {
            displayName: 'CVE',
            searchFilterName: 'CVE',
        },
        {
            displayName: 'Severity',
            searchFilterName: 'SEVERITY',
        },
        {
            displayName: 'Status',
            searchFilterName: 'FIXABLE',
        },
    ];

    return (
        <Toolbar>
            <ToolbarContent>
                <FilterAutocomplete
                    searchFilter={searchFilter}
                    onFilterChange={(newFilter, { action, category, value }) => {
                        setSearchFilter(newFilter);
                        if (action === 'ADD') {
                            trackAppliedFilter(category, value);
                        }
                    }}
                    searchOptions={searchOptions}
                    autocompleteSearchContext={autocompleteSearchContext}
                />
                <ToolbarGroup>
                    <CVESeverityDropdown searchFilter={searchFilter} onSelect={onSelect} />
                    <CVEStatusDropdown searchFilter={searchFilter} onSelect={onSelect} />
                </ToolbarGroup>
                <ToolbarGroup aria-label="applied search filters" className="pf-u-w-100">
                    <SearchFilterChips
                        onFilterChange={onFilterChange}
                        filterChipGroupDescriptors={filterChipGroupDescriptors}
                    />
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default NodeCveFilterToolbar;

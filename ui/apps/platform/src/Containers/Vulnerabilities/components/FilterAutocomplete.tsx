import React, { useState, useMemo } from 'react';
import { debounce, Select, SelectOption, Skeleton, ToolbarGroup } from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import { SearchFilter } from 'types/search';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { searchValueAsArray } from 'utils/searchUtils';
import SearchOptionsDropdown from './SearchOptionsDropdown';
import { applyRegexSearchModifiers } from '../WorkloadCves/searchUtils';
import { SearchOption, SearchOptionValue, regexSearchOptions } from '../searchOptions';

import './FilterAutocomplete.css';

function getOptions(data: string[] | undefined): React.ReactElement[] {
    return data?.map((value) => <SelectOption key={value} value={value} />) ?? [];
}

function getAutocompleteOptionsQueryString(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .map(([key, value]) => {
            let returnValue = '';
            if (value) {
                returnValue = `${Array.isArray(value) ? value.join(',') : value}`;
            }
            return `${key}:${returnValue}`;
        })
        .join('+');
}

export type FilterChangeEvent = {
    action: 'ADD' | 'REMOVE';
    category: SearchOptionValue;
    value: string;
};

export type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter, changeEvent: FilterChangeEvent) => void;
    searchOptions: SearchOption[];
    autocompleteSearchContext?:
        | { 'Image SHA': string }
        | { 'Deployment ID': string }
        | { 'CVE ID': string }
        | Record<string, never>;
};

function FilterAutocompleteSelect({
    searchFilter,
    onFilterChange,
    searchOptions,
    autocompleteSearchContext = {},
}: FilterAutocompleteSelectProps) {
    const [searchOption, setSearchOption] = useState<SearchOption>(() => {
        return searchOptions[0];
    });
    const [typeahead, setTypeahead] = useState('');
    const [isTyping, setIsTyping] = useState(false);
    const { isOpen, onToggle } = useSelectToggle();

    // Autocomplete requests for "Cluster" never return results if there is a 'CVE ID', 'Severity', or 'Fixable' search filter
    // included in the query. In this case we don't include the additional filters at all which leaves the cluster results
    // unfiltered. Not ideal, but better than no results.
    const useSearchContextForAutocomplete =
        searchOption.value !== 'CLUSTER' ||
        (!autocompleteSearchContext['CVE ID'] && !searchFilter.SEVERITY && !searchFilter.FIXABLE);

    const autocompleteSearchFilterBase = useSearchContextForAutocomplete
        ? { ...autocompleteSearchContext, ...searchFilter }
        : {};

    // If we are using regex matching, apply the regex modifier to the search filter
    const autocompleteSearchFilter = applyRegexSearchModifiers(autocompleteSearchFilterBase);

    // Append the current typeahead value to the search filter, use regex matching only if:
    // 1. The typeahead is not empty
    // 2. The search option supports regex matching
    autocompleteSearchFilter[searchOption.value] =
        typeahead !== '' && regexSearchOptions.some((option) => option === searchOption.value)
            ? [`r/${typeahead}`]
            : [typeahead];

    const variables = {
        query: getAutocompleteOptionsQueryString(autocompleteSearchFilter),
        categories: searchOption.category,
    };

    const { data, loading } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onSelect(value) {
        setTypeahead('');

        const category = searchOption.value;
        const oldValues = searchValueAsArray(searchFilter[category]);
        const action = oldValues.includes(value) ? 'REMOVE' : 'ADD';

        const newValues =
            action === 'ADD'
                ? oldValues.concat(value)
                : oldValues.filter((f: string) => f !== value);

        onFilterChange({ ...searchFilter, [category]: newValues }, { action, category, value });
    }

    // Debounce the autocomplete requests to not overload the backend
    const updateTypeahead = useMemo(
        () =>
            debounce((value: string) => {
                setTypeahead(value);
                setIsTyping(false);
            }, 800),
        []
    );

    function getSuggestedOptions() {
        if (loading || isTyping) {
            return [
                <SelectOption
                    key="autocomplete-options-loading"
                    value="autocomplete-options-loading"
                >
                    <Skeleton screenreaderText="Loading suggested options" />
                </SelectOption>,
            ];
        }
        return getOptions(data?.searchAutocomplete);
    }

    return (
        <ToolbarGroup
            variant="filter-group"
            className="pf-u-display-flex pf-u-flex-grow-1"
            id="filter-autocomplete-toolbar-group"
        >
            <SearchOptionsDropdown
                setSearchOption={(selection) => {
                    const newSearchOption = searchOptions.find(
                        (option) => option.value === selection
                    );
                    if (newSearchOption) {
                        setTypeahead('');
                        setSearchOption(newSearchOption);
                    }
                }}
                searchOption={searchOption}
            >
                {searchOptions.map(({ label, value }) => {
                    return (
                        <SelectOption key={label} value={value}>
                            {label}
                        </SelectOption>
                    );
                })}
            </SearchOptionsDropdown>
            <Select
                typeAheadAriaLabel={`Filter by ${searchOption.label}`}
                aria-label={`Filter by ${searchOption.label}`}
                onSelect={(e, value) => {
                    onSelect(value);
                }}
                onToggle={onToggle}
                isOpen={isOpen}
                placeholderText={`Filter results by ${searchOption.label}`}
                variant="typeaheadmulti"
                isCreatable
                createText="Add"
                onFilter={getSuggestedOptions}
                // We set this as empty because we want to use SearchFilterChips to display the search values
                selections={searchFilter[searchOption.value]}
                onTypeaheadInputChanged={(val: string) => {
                    setIsTyping(true);
                    updateTypeahead(val);
                }}
                className="pf-u-flex-grow-1"
            >
                {getSuggestedOptions()}
            </Select>
        </ToolbarGroup>
    );
}

export default FilterAutocompleteSelect;

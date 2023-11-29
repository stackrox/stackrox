import React, { useState, useMemo } from 'react';
import { debounce, Select, SelectOption, Skeleton, ToolbarGroup } from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import { SearchFilter } from 'types/search';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import SearchOptionsDropdown from './SearchOptionsDropdown';
import { SearchOption } from '../searchOptions';

import './FilterAutocomplete.css';

function getOptions(data: string[] | undefined): React.ReactElement[] | undefined {
    return data?.map((value) => <SelectOption key={value} value={value} />);
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

export type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
    searchOptions: SearchOption[];
    autocompleteSearchContext?:
        | { 'Image SHA': string }
        | { 'Deployment ID': string }
        | { 'CVE ID': string }
        | Record<string, never>;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
    searchOptions,
    autocompleteSearchContext = {},
}: FilterAutocompleteSelectProps) {
    const [searchOption, setSearchOption] = useState<SearchOption>(() => {
        return searchOptions[0];
    });
    const [typeahead, setTypeahead] = useState('');
    const [isTyping, setIsTyping] = useState(false);
    const { isOpen, onToggle } = useSelectToggle();

    // TODO Autocomplete requests for "Cluster" never return results if there is a 'CVE ID' or 'Severity' search filter
    // included in the query. In this case we don't include the additional filters at all which leaves the cluster results
    // unfiltered. Not ideal, but better than no results.
    const autocompleteSearchFilter =
        searchOption.value === 'CLUSTER' && autocompleteSearchContext['CVE ID']
            ? { [searchOption.value]: typeahead }
            : {
                  ...autocompleteSearchContext,
                  ...searchFilter,
                  [searchOption.value]: typeahead,
              };

    const variables = {
        query: getAutocompleteOptionsQueryString(autocompleteSearchFilter),
        categories: searchOption.category,
    };

    const { data, loading } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onSelect(newValue) {
        const oldValue = searchFilter[searchOption.value] as string[];
        setTypeahead('');
        if (oldValue?.includes(newValue)) {
            setSearchFilter({
                ...searchFilter,
                [searchOption.value]: oldValue.filter((fil: string) => fil !== newValue),
            });
        } else {
            setSearchFilter({
                ...searchFilter,
                [searchOption.value]: oldValue ? [...oldValue, newValue] : [newValue],
            });
        }
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

    const autocompleteOptions = loading || isTyping ? [] : getOptions(data?.searchAutocomplete);

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
                // We set this as empty because we want to use SearchFilterChips to display the search values
                selections={searchFilter[searchOption.value]}
                onTypeaheadInputChanged={(val: string) => {
                    setIsTyping(true);
                    updateTypeahead(val);
                }}
                className="pf-u-flex-grow-1"
            >
                {autocompleteOptions}
            </Select>
        </ToolbarGroup>
    );
}

export default FilterAutocompleteSelect;

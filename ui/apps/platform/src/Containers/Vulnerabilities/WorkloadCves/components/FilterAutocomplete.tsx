import React, { useState, useMemo } from 'react';
import { debounce, Select, SelectOption, ToolbarGroup } from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import { SearchFilter } from 'types/search';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { searchCategories } from 'constants/entityTypes';
import FilterResourceDropdown, {
    FilterResourceDropdownProps,
    Resource,
    resources,
} from './FilterResourceDropdown';
import { parseQuerySearchFilter } from '../searchUtils';

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

function getSearchCategoriesForAutocomplete(resource: Resource) {
    if (resource === 'CVE') {
        return searchCategories.IMAGE_CVE;
    }
    return searchCategories[resource];
}

export type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
    supportedResourceFilters?: FilterResourceDropdownProps['supportedResourceFilters'];
    autocompleteSearchContext?:
        | { 'Image SHA': string }
        | { 'Deployment ID': string }
        | { 'CVE ID': string }
        | Record<string, never>;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
    supportedResourceFilters,
    autocompleteSearchContext = {},
}: FilterAutocompleteSelectProps) {
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [resource, setResource] = useState<Resource>(
        () => resources.find((r) => supportedResourceFilters?.has(r)) ?? 'DEPLOYMENT'
    );
    const [typeahead, setTypeahead] = useState('');
    const { isOpen, onToggle } = useSelectToggle();

    // TODO Autocomplete requests for "Cluster" never return results if there is a 'CVE ID' or 'Severity' search filter
    // included in the query. In this case we don't include the additional filters at all which leaves the cluster results
    // unfiltered. Not ideal, but better than no results.
    const autocompleteSearchFilter =
        resource === 'CLUSTER' && autocompleteSearchContext['CVE ID']
            ? { [resource]: typeahead }
            : { ...autocompleteSearchContext, ...querySearchFilter, [resource]: typeahead };

    const variables = {
        query: getAutocompleteOptionsQueryString(autocompleteSearchFilter),
        categories: getSearchCategoriesForAutocomplete(resource),
    };

    const { data } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onSelect(newValue) {
        const oldValue = searchFilter[resource] as string[];
        setTypeahead('');
        if (oldValue?.includes(newValue)) {
            setSearchFilter({
                ...searchFilter,
                [resource]: oldValue.filter((fil: string) => fil !== newValue),
            });
        } else {
            setSearchFilter({
                ...searchFilter,
                [resource]: oldValue ? [...oldValue, newValue] : [newValue],
            });
        }
    }

    // Debounce the autocomplete requests to not overload the backend
    const updateTypeahead = useMemo(
        () => debounce((value: string) => setTypeahead(value), 800),
        []
    );

    return (
        <ToolbarGroup variant="filter-group" className="pf-u-display-flex pf-u-flex-grow-1">
            <FilterResourceDropdown
                setResource={setResource}
                resource={resource}
                supportedResourceFilters={supportedResourceFilters}
            />
            <Select
                typeAheadAriaLabel={`Filter by ${resource}`}
                aria-label={`Filter by ${resource}`}
                onSelect={(e, value) => {
                    onSelect(value);
                }}
                onToggle={onToggle}
                isOpen={isOpen}
                placeholderText={`Filter results by ${resource.toLowerCase()}`}
                variant="typeaheadmulti"
                isCreatable
                createText="Add"
                selections={searchFilter[resource]}
                onTypeaheadInputChanged={(val: string) => {
                    updateTypeahead(val);
                }}
                className="pf-u-flex-grow-1"
            >
                {getOptions(data?.searchAutocomplete)}
            </Select>
        </ToolbarGroup>
    );
}

export default FilterAutocompleteSelect;

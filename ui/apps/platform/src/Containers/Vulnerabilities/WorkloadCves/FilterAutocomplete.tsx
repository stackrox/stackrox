import React, { useState, useMemo } from 'react';
import { debounce, Select, SelectOption, ToolbarGroup } from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import { SearchFilter } from 'types/search';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { searchCategories } from 'constants/entityTypes';
import FilterResourceDropdown, { Resource } from './FilterResourceDropdown';

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

// we have to convert IMAGE_CVE to CVE for the query to populate
// autocomplete for Image CVEs
function getResourceQueryString(resource: Resource) {
    if (resource === 'IMAGE_CVE') {
        return 'CVE';
    }
    return resource;
}

type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
    resourceContext?: Resource;
    onDeleteGroup: (category) => void;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
    resourceContext,
    onDeleteGroup,
}: FilterAutocompleteSelectProps) {
    const [resource, setResource] = useState<Resource>('DEPLOYMENT');
    const [typeahead, setTypeahead] = useState(searchFilter[resource] || '');
    const { isOpen, onToggle } = useSelectToggle();
    const variables = {
        query: getAutocompleteOptionsQueryString({ [getResourceQueryString(resource)]: typeahead }),
        categories: searchCategories[resource],
    };

    const { data } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onSelect(newValue) {
        const oldValue = searchFilter[resource] as string[];
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
        <ToolbarGroup variant="filter-group" className="pf-u-flex-grow-1">
            <FilterResourceDropdown
                setResource={setResource}
                resource={resource}
                resourceContext={resourceContext}
            />
            <Select
                aria-label={`Filter by ${resource as string}`}
                onSelect={(e, value) => {
                    onSelect(value);
                }}
                onClear={() => onDeleteGroup(resource)}
                onToggle={onToggle}
                isOpen={isOpen}
                placeholder={`Filter by ${resource as string}`}
                variant="typeaheadmulti"
                isCreatable
                createText="Add"
                selections={searchFilter[resource]}
                onTypeaheadInputChanged={(val: string) => {
                    updateTypeahead(val);
                }}
                className="pf-u-w-100"
            >
                {getOptions(data?.searchAutocomplete)}
            </Select>
        </ToolbarGroup>
    );
}

export default FilterAutocompleteSelect;

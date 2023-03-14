import React, { useState, useCallback, useMemo } from 'react';
import {
    debounce,
    Select,
    SelectOption,
    ToolbarFilter,
    ToolbarGroup,
} from '@patternfly/react-core';
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

type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
    resourceContext?: Resource;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
    resourceContext,
}: FilterAutocompleteSelectProps) {
    const [resource, setResource] = useState<Resource>('DEPLOYMENT');
    const [typeahead, setTypeahead] = useState(searchFilter[resource] || '');
    const { isOpen, onToggle } = useSelectToggle();
    const variables = {
        query: getAutocompleteOptionsQueryString({ [resource]: typeahead }),
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

    function onDelete(category: Resource, id: string) {
        const newSearchFilter = { ...searchFilter };
        const newResourceFilter = searchFilter[category] as string[];
        newSearchFilter[category] = newResourceFilter.filter((fil: string) => fil !== id);
        setSearchFilter(newSearchFilter);
    }

    function onDeleteGroup(category: Resource) {
        const newSearchFilter = { ...searchFilter };
        delete newSearchFilter[category];
        setSearchFilter(newSearchFilter);
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
            <ToolbarFilter
                chips={searchFilter.DEPLOYMENT ? (searchFilter.DEPLOYMENT as string[]) : []}
                deleteChip={(_, chip) => onDelete('DEPLOYMENT', chip as string)}
                deleteChipGroup={() => onDeleteGroup('DEPLOYMENT')}
                categoryName="Deployment"
            />
            <ToolbarFilter
                chips={searchFilter.CVE ? (searchFilter.CVE as string[]) : []}
                deleteChip={(category, chip) => onDelete(category as Resource, chip as string)}
                deleteChipGroup={(category) => onDeleteGroup(category as Resource)}
                categoryName="CVE"
            />
            <ToolbarFilter
                chips={searchFilter.IMAGE ? (searchFilter.IMAGE as string[]) : []}
                deleteChip={(_, chip) => onDelete('IMAGE', chip as string)}
                deleteChipGroup={() => onDeleteGroup('IMAGE')}
                categoryName="Image"
            />
            <ToolbarFilter
                chips={searchFilter.NAMESPACE ? (searchFilter.NAMESPACE as string[]) : []}
                deleteChip={(_, chip) => onDelete('NAMESPACE', chip as string)}
                deleteChipGroup={() => onDeleteGroup('NAMESPACE')}
                categoryName="Namespace"
            />
            <ToolbarFilter
                chips={searchFilter.CLUSTER ? (searchFilter.CLUSTER as string[]) : []}
                deleteChip={(_, chip) => onDelete('CLUSTER', chip as string)}
                deleteChipGroup={() => onDeleteGroup('CLUSTER')}
                categoryName="Cluster"
            />
        </ToolbarGroup>
    );
}

export default FilterAutocompleteSelect;

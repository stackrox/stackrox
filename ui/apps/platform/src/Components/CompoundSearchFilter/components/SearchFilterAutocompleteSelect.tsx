import { useState } from 'react';
import { Button } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import type { SearchFilter } from 'types/search';

import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';

import AutocompleteSelect from './AutocompleteSelect';

export type SearchFilterAutocompleteSelectProps = {
    additionalContextFilter?: SearchFilter;
    attribute: GenericSearchFilterAttribute;
    onSearch: OnSearchCallback;
    searchCategory: string;
    searchFilter: SearchFilter;
};

function SearchFilterAutocompleteSelect({
    additionalContextFilter,
    attribute,
    onSearch,
    searchCategory,
    searchFilter,
}: SearchFilterAutocompleteSelectProps) {
    const { filterChipLabel, searchTerm } = attribute;
    const textLabel = `Filter results by ${filterChipLabel}`;

    const [value, setValue] = useState('');

    const handleSearch = (valueSelected: string) => {
        onSearch([
            {
                action: 'APPEND',
                category: searchTerm,
                value: valueSelected,
            },
        ]);
        setValue('');
    };

    return (
        <>
            <AutocompleteSelect
                searchCategory={searchCategory}
                searchTerm={searchTerm}
                value={value}
                onChange={setValue}
                onSearch={handleSearch}
                textLabel={textLabel}
                searchFilter={searchFilter}
                additionalContextFilter={additionalContextFilter}
            />
            <Button
                variant="control"
                aria-label="Apply autocomplete input to search"
                onClick={() => handleSearch(value)}
            >
                <ArrowRightIcon />
            </Button>
        </>
    );
}

export default SearchFilterAutocompleteSelect;

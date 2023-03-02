import React from 'react';
import { SearchInput } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

type FilterAutocompleteInputProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
};

function FilterAutocompleteInput({ searchFilter, setSearchFilter }: FilterAutocompleteInputProps) {
    function onInputChange(newValue: string) {
        setSearchFilter({
            ...searchFilter,
            id: newValue,
        });
    }

    return (
        <SearchInput
            aria-label="filter by CVE ID"
            onChange={(e, value) => {
                onInputChange(value);
            }}
            value={(searchFilter.id as string) || ''}
            onClear={() => {
                onInputChange('');
            }}
            placeholder="Filter by CVE ID"
        />
    );
}

export default FilterAutocompleteInput;

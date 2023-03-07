import React from 'react';
import { SearchInput } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';

type FilterAutocompleteSelectProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (s) => void;
};

function FilterAutocompleteSelect({
    searchFilter,
    setSearchFilter,
}: FilterAutocompleteSelectProps) {
    function onInputChange(newValue: string) {
        setSearchFilter({
            ...searchFilter,
            id: newValue,
        });
    }

    const { resource } = searchFilter;

    return (
        <SearchInput
            aria-label={`Filter by ${resource}`}
            onChange={(e, value) => {
                onInputChange(value);
            }}
            value={(searchFilter.id as string) || ''}
            onClear={() => {
                onInputChange('');
            }}
            placeholder={`Filter by ${resource}`}
        />
    );
}

export default FilterAutocompleteSelect;

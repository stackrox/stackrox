import React, { useState } from 'react';
import { SearchInput } from '@patternfly/react-core';

type SearchFilterNamesProps = {
    namesSelected: string[] | undefined;
    isDisabled: boolean;
    setNamesSelected: (names: string[]) => void;
};

function SearchFilterNames({
    namesSelected,
    isDisabled,
    setNamesSelected,
}: SearchFilterNamesProps) {
    const [searchValue, setSearchValue] = useState('');

    function onSearchInputChange(_event, value) {
        setSearchValue(value);
    }

    function onSearch() {
        const previousNames = namesSelected ?? [];
        const nextNames = previousNames.includes(searchValue)
            ? previousNames
            : [...previousNames, searchValue];

        setNamesSelected(nextNames);
        setSearchValue('');
    }

    return (
        <SearchInput
            aria-label="Filter by name"
            placeholder="Filter by name"
            isDisabled={isDisabled}
            value={searchValue}
            onChange={onSearchInputChange}
            onSearch={onSearch}
            onClear={() => setSearchValue('')}
        />
    );
}

export default SearchFilterNames;

import type { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';

import type { OnSearchCallback, SelectExclusiveSingleSearchFilterAttribute } from '../types';

export type SearchFilterSelectExclusiveSingleProps = {
    attribute: SelectExclusiveSingleSearchFilterAttribute;
    isSeparate?: boolean; // default false if within CompoundSearchFilter
    onSearch: OnSearchCallback;
    searchFilter: SearchFilter;
};

function SearchFilterSelectExclusiveSingle({
    attribute,
    isSeparate = false,
    onSearch,
    searchFilter,
}: SearchFilterSelectExclusiveSingleProps): ReactElement {
    const { displayName, inputProps, searchTerm: category } = attribute;
    const { options } = inputProps;
    const placeholderText = isSeparate ? displayName : `Filter by ${displayName}`;

    function handleSelect(_id, value: string) {
        onSearch([
            {
                action: 'SELECT_EXCLUSIVE',
                category,
                value,
            },
        ]);
    }

    const values = searchValueAsArray(searchFilter[category]);
    const value = values.length === 1 ? values[0] : '';

    return (
        <SelectSingle
            id={category}
            isFullWidth={false}
            placeholderText={placeholderText}
            toggleAriaLabel={`${placeholderText} select menu`}
            value={value}
            handleSelect={handleSelect}
        >
            {options.map(({ label, value }) => (
                <SelectOption key={value} value={value}>
                    {label}
                </SelectOption>
            ))}
        </SelectSingle>
    );
}

export default SearchFilterSelectExclusiveSingle;

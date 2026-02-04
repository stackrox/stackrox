import type { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';
import type { SearchFilter } from 'types/search';
import { searchValueAsArray } from 'utils/searchUtils';
import type { NonEmptyArray } from 'utils/type.utils';

import type {
    OnSearchCallback,
    OnSearchPayloadItem,
    SelectExclusiveDoubleSearchFilterAttribute,
    SelectExclusiveDoubleSearchFilterOption,
} from '../types';

function getPayloadItemsForCategoriesAndOption(
    category1: string,
    category2: string,
    option: SelectExclusiveDoubleSearchFilterOption
): NonEmptyArray<OnSearchPayloadItem> {
    const { category, value } = option;

    // Select category and value for the option and delete any value for the other category.
    return [
        {
            action: 'SELECT_EXCLUSIVE',
            category,
            value,
        },
        {
            action: 'DELETE',
            category: category !== category1 ? category1 : category2,
        },
    ];
}

function getLabelOfSelectedOption(
    category1: string,
    category2: string,
    options: NonEmptyArray<SelectExclusiveDoubleSearchFilterOption>,
    searchFilter: SearchFilter
): string {
    const values1 = searchValueAsArray(searchFilter[category1]);
    const values2 = searchValueAsArray(searchFilter[category2]);
    const value1 = values1.length === 1 ? values1[0] : '';
    const value2 = values2.length === 1 ? values2[0] : '';

    // Both values non-empty implies inconsistent state from untrusted query string of URL.
    if ((value1 !== '' && value2 === '') || (value1 === '' && value2 !== '')) {
        const option = options.find(
            ({ category, value }) =>
                (category === category1 && value === value1) ||
                (category === category2 && value === value2)
        );

        if (option) {
            return option.label;
        }
    }

    return '';
}

export type SearchFilterSelectExclusiveDoubleProps = {
    attribute: SelectExclusiveDoubleSearchFilterAttribute;
    isSeparate?: boolean; // default false if within CompoundSearchFilter
    onSearch: OnSearchCallback;
    searchFilter: SearchFilter;
};

function SearchFilterSelectExclusiveDouble({
    attribute,
    isSeparate = false,
    onSearch,
    searchFilter,
}: SearchFilterSelectExclusiveDoubleProps): ReactElement {
    const { displayName, inputProps, searchTerm: category1 } = attribute;
    const { category2, options } = inputProps;
    const placeholderText = isSeparate ? displayName : `Filter by ${displayName}`;

    function handleSelect(_id, labelSelected: string) {
        const option = options.find(({ label }) => label === labelSelected);
        if (option) {
            onSearch(getPayloadItemsForCategoriesAndOption(category1, category2, option));
        }
    }

    const labelOfSelectedOption = getLabelOfSelectedOption(
        category1,
        category2,
        options,
        searchFilter
    );

    return (
        <SelectSingle
            id={category1}
            isFullWidth={false}
            placeholderText={placeholderText}
            toggleAriaLabel={`${placeholderText} select menu`}
            value={labelOfSelectedOption}
            handleSelect={handleSelect}
        >
            {options.map(({ label }) => (
                <SelectOption key={label} value={label}>
                    {label}
                </SelectOption>
            ))}
        </SelectSingle>
    );
}

export default SearchFilterSelectExclusiveDouble;

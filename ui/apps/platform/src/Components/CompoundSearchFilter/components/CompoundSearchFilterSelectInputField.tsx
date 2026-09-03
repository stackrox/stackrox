import type { SearchFilter } from 'types/search';
import { ensureExhaustive } from 'utils/type.utils';

import type { GenericSelectSearchFilterAttribute, OnSearchCallback } from '../types';
import SearchFilterSelectExclusiveDouble from './SearchFilterSelectExclusiveDouble';
import SearchFilterSelectExclusiveSingle from './SearchFilterSelectExclusiveSingle';
import SearchFilterSelectInclusive from './SearchFilterSelectInclusive';

export type CompoundSearchFilterSelectInputFieldProps = {
    attribute: GenericSelectSearchFilterAttribute;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
    searchFilter: SearchFilter;
};

function CompoundSearchFilterSelectInputField({
    attribute,
    isDisabled = false,
    onSearch,
    searchFilter,
}: CompoundSearchFilterSelectInputFieldProps) {
    const { inputType } = attribute;
    switch (inputType) {
        case 'select-exclusive-double':
            return (
                <SearchFilterSelectExclusiveDouble
                    attribute={attribute}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        case 'select-exclusive-single':
            return (
                <SearchFilterSelectExclusiveSingle
                    attribute={attribute}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        case 'select':
            return (
                <SearchFilterSelectInclusive
                    attribute={attribute}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        default:
            return ensureExhaustive(inputType);
    }
}

export default CompoundSearchFilterSelectInputField;

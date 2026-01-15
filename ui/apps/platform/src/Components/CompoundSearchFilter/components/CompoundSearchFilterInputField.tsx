import type { SearchFilter } from 'types/search';
import { ensureExhaustive } from 'utils/type.utils';

import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterEntity,
    OnSearchCallback,
} from '../types';
import SearchFilterConditionDate from './SearchFilterConditionDate';
import SearchFilterConditionNumber from './SearchFilterConditionNumber';
import SearchFilterConditionText from './SearchFilterConditionText';
import SearchFilterAutocompleteSelect from './SearchFilterAutocompleteSelect';
import SearchFilterSelectExclusiveDouble from './SearchFilterSelectExclusiveDouble';
import SearchFilterSelectExclusiveSingle from './SearchFilterSelectExclusiveSingle';
import SearchFilterSelectInclusive from './SearchFilterSelectInclusive';
import SearchFilterText from './SearchFilterText';

export type CompoundSearchFilterInputFieldProps = {
    entity: CompoundSearchFilterEntity;
    attribute: CompoundSearchFilterAttribute;
    onSearch: OnSearchCallback;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
};

function CompoundSearchFilterInputField({
    entity,
    attribute,
    onSearch,
    searchFilter,
    additionalContextFilter,
}: CompoundSearchFilterInputFieldProps) {
    const { inputType } = attribute;
    switch (inputType) {
        case 'text':
            return <SearchFilterText attribute={attribute} onSearch={onSearch} />;
        case 'date-picker':
            return <SearchFilterConditionDate attribute={attribute} onSearch={onSearch} />;
        case 'condition-number':
            return <SearchFilterConditionNumber attribute={attribute} onSearch={onSearch} />;
        case 'condition-text':
            return <SearchFilterConditionText attribute={attribute} onSearch={onSearch} />;
        case 'autocomplete':
            return (
                <SearchFilterAutocompleteSelect
                    additionalContextFilter={additionalContextFilter}
                    attribute={attribute}
                    onSearch={onSearch}
                    searchCategory={entity.searchCategory}
                    searchFilter={searchFilter}
                />
            );
        case 'select-exclusive-double':
            return (
                <SearchFilterSelectExclusiveDouble
                    attribute={attribute}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        case 'select-exclusive-single':
            return (
                <SearchFilterSelectExclusiveSingle
                    attribute={attribute}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        case 'select':
            return (
                <SearchFilterSelectInclusive
                    attribute={attribute}
                    onSearch={onSearch}
                    searchFilter={searchFilter}
                />
            );
        case 'unspecified':
            // placeholder because not for compound search filter but only for certain attributes in view-based report
            // For example, Image CVE discovered time: All time
            return <></>;
        default:
            return ensureExhaustive(inputType);
    }
}

export default CompoundSearchFilterInputField;

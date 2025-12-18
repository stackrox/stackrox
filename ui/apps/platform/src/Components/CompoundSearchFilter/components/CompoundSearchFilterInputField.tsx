import type { SearchFilter } from 'types/search';
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
    if (attribute.inputType === 'text') {
        return <SearchFilterText attribute={attribute} onSearch={onSearch} />;
    }
    if (attribute.inputType === 'date-picker') {
        return <SearchFilterConditionDate attribute={attribute} onSearch={onSearch} />;
    }
    if (attribute.inputType === 'condition-number') {
        return <SearchFilterConditionNumber attribute={attribute} onSearch={onSearch} />;
    }
    if (attribute.inputType === 'condition-text') {
        return <SearchFilterConditionText attribute={attribute} onSearch={onSearch} />;
    }
    if (attribute.inputType === 'autocomplete') {
        return (
            <SearchFilterAutocompleteSelect
                additionalContextFilter={additionalContextFilter}
                attribute={attribute}
                onSearch={onSearch}
                searchCategory={entity.searchCategory}
                searchFilter={searchFilter}
            />
        );
    }
    if (attribute.inputType === 'select-exclusive-double') {
        return (
            <SearchFilterSelectExclusiveDouble
                attribute={attribute}
                onSearch={onSearch}
                searchFilter={searchFilter}
            />
        );
    }
    if (attribute.inputType === 'select-exclusive-single') {
        return (
            <SearchFilterSelectExclusiveSingle
                attribute={attribute}
                onSearch={onSearch}
                searchFilter={searchFilter}
            />
        );
    }
    if (attribute.inputType === 'select') {
        return (
            <SearchFilterSelectInclusive
                attribute={attribute}
                onSearch={onSearch}
                searchFilter={searchFilter}
            />
        );
    }
    return <div />;
}

export default CompoundSearchFilterInputField;

import { Button } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import type { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterEntity,
    OnSearchCallback,
} from '../types';
import SearchFilterConditionDate from './SearchFilterConditionDate';
import SearchFilterConditionNumber from './SearchFilterConditionNumber';
import SearchFilterConditionText from './SearchFilterConditionText';
import SearchFilterAutocomplete from './SearchFilterAutocomplete';
import SearchFilterSelectExclusiveDouble from './SearchFilterSelectExclusiveDouble';
import SearchFilterSelectExclusiveSingle from './SearchFilterSelectExclusiveSingle';
import SearchFilterSelectInclusive from './SearchFilterSelectInclusive';
import SearchFilterText from './SearchFilterText';

export type InputFieldValue =
    | string
    | number
    | undefined
    | string[]
    | { condition: string; number: number }
    | { condition: string; date: string };
export type InputFieldOnChange = (value: InputFieldValue) => void;

export type CompoundSearchFilterInputFieldProps = {
    entity: CompoundSearchFilterEntity;
    attribute: CompoundSearchFilterAttribute;
    value: InputFieldValue;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
    onSearch: OnSearchCallback;
    onChange: InputFieldOnChange;
};

function CompoundSearchFilterInputField({
    entity,
    attribute,
    value,
    searchFilter,
    additionalContextFilter,
    onSearch,
    onChange,
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
        const { searchCategory } = entity;
        const { searchTerm, filterChipLabel } = attribute;
        const textLabel = `Filter results by ${filterChipLabel}`;

        const handleSearch = (newValue: string) => {
            onSearch([
                {
                    action: 'APPEND',
                    category: attribute.searchTerm,
                    value: newValue,
                },
            ]);
            onChange('');
        };
        return (
            <>
                <SearchFilterAutocomplete
                    searchCategory={searchCategory}
                    searchTerm={searchTerm}
                    value={ensureString(value)}
                    onChange={(newValue) => {
                        onChange(newValue);
                    }}
                    onSearch={handleSearch}
                    textLabel={textLabel}
                    searchFilter={searchFilter}
                    additionalContextFilter={additionalContextFilter}
                />
                <Button
                    variant="control"
                    aria-label="Apply autocomplete input to search"
                    onClick={() => handleSearch(ensureString(value))}
                >
                    <ArrowRightIcon />
                </Button>
            </>
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

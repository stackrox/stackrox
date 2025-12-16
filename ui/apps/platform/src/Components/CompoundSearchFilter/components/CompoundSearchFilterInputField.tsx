import { Button } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import type { SearchFilter } from 'types/search';
import { ensureString } from 'utils/ensure';
import type { SelectedEntity } from './EntitySelector';
import type { SelectedAttribute } from './AttributeSelector';
import type { CompoundSearchFilterConfig, OnSearchCallback } from '../types';
import {
    conditionMap,
    dateConditionMap,
    ensureConditionDate,
    ensureConditionNumber,
    getAttribute,
    getEntity,
} from '../utils/utils';
import ConditionNumber from './ConditionNumber';
import SearchFilterAutocomplete from './SearchFilterAutocomplete';
import SearchFilterSelectExclusiveDouble from './SearchFilterSelectExclusiveDouble';
import SearchFilterSelectExclusiveSingle from './SearchFilterSelectExclusiveSingle';
import SearchFilterInclusive from './SearchFilterSelectInclusive';
import SearchFilterText from './SearchFilterText';
import ConditionDate from './ConditionDate';
import ConditionText from './ConditionText';

export type InputFieldValue =
    | string
    | number
    | undefined
    | string[]
    | { condition: string; number: number }
    | { condition: string; date: string };
export type InputFieldOnChange = (value: InputFieldValue) => void;

export type CompoundSearchFilterInputFieldProps = {
    selectedEntity: SelectedEntity;
    selectedAttribute: SelectedAttribute;
    value: InputFieldValue;
    searchFilter: SearchFilter;
    additionalContextFilter?: SearchFilter;
    onSearch: OnSearchCallback;
    onChange: InputFieldOnChange;
    config: CompoundSearchFilterConfig;
};

function CompoundSearchFilterInputField({
    selectedEntity,
    selectedAttribute,
    value,
    searchFilter,
    additionalContextFilter,
    onSearch,
    onChange,
    config,
}: CompoundSearchFilterInputFieldProps) {
    if (!selectedEntity || !selectedAttribute) {
        return null;
    }

    const entity = getEntity(config, selectedEntity);
    const attribute = getAttribute(config, selectedEntity, selectedAttribute);

    if (!attribute) {
        return null;
    }

    if (attribute.inputType === 'text') {
        return <SearchFilterText attribute={attribute} onSearch={onSearch} />;
    }
    if (attribute.inputType === 'date-picker') {
        return (
            <ConditionDate
                value={ensureConditionDate(value)}
                onChange={(newValue) => {
                    onChange(newValue);
                }}
                onSearch={(newValue) => {
                    const { condition, date } = newValue;
                    onSearch([
                        {
                            action: 'APPEND',
                            category: attribute.searchTerm,
                            value: `${dateConditionMap[condition]}${date}`,
                        },
                    ]);
                    onChange({ ...newValue, date: '' });
                }}
            />
        );
    }
    if (attribute.inputType === 'condition-number') {
        return (
            <ConditionNumber
                value={ensureConditionNumber(value)}
                onChange={(newValue) => {
                    onChange(newValue);
                }}
                onSearch={(newValue) => {
                    const { condition, number } = newValue;
                    onChange(newValue);
                    onSearch([
                        {
                            action: 'APPEND',
                            category: attribute.searchTerm,
                            value: `${conditionMap[condition]}${number}`,
                        },
                    ]);
                }}
            />
        );
    }
    if (attribute.inputType === 'condition-text') {
        return (
            <ConditionText
                inputProps={attribute.inputProps}
                onSearch={(internalConditionText) => {
                    // onChange(newValue); // inputText seems unused in CompoundSearchFilter
                    onSearch([
                        {
                            action: 'APPEND',
                            category: attribute.searchTerm,
                            value: internalConditionText,
                        },
                    ]);
                }}
            />
        );
    }
    if (entity && entity.searchCategory && attribute.inputType === 'autocomplete') {
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
            <SearchFilterInclusive
                attribute={attribute}
                onSearch={onSearch}
                searchFilter={searchFilter}
            />
        );
    }
    return <div />;
}

export default CompoundSearchFilterInputField;

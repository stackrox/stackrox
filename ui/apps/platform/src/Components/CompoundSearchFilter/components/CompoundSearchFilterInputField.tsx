import React from 'react';
import { DatePicker, SearchInput, SelectOption } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';
import { getDate } from 'utils/dateUtils';
import { dateFormat } from 'constants/dateTimeFormat';
import { SelectedEntity } from './EntitySelector';
import { SelectedAttribute } from './AttributeSelector';
import {
    OnSearchPayload,
    PartialCompoundSearchFilterConfig,
    SearchFilterAttribute,
    SelectSearchFilterAttribute,
} from '../types';
import { ensureConditionNumber, ensureString, ensureStringArray } from '../utils/utils';

import CheckboxSelect from './CheckboxSelect';
import ConditionNumber, { conditionMap } from './ConditionNumber';
import SearchFilterAutocomplete from './SearchFilterAutocomplete';

export type InputFieldValue =
    | string
    | number
    | undefined
    | string[]
    | { condition: string; number: number };
export type InputFieldOnChange = (value: InputFieldValue) => void;

export type CompoundSearchFilterInputFieldProps = {
    selectedEntity: SelectedEntity;
    selectedAttribute: SelectedAttribute;
    value: InputFieldValue;
    searchFilter: SearchFilter;
    onSearch: ({ action, category, value }: OnSearchPayload) => void;
    onChange: InputFieldOnChange;
    config: PartialCompoundSearchFilterConfig;
};

function isSelectType(
    attributeObject: SearchFilterAttribute
): attributeObject is SelectSearchFilterAttribute {
    return attributeObject.inputType === 'select';
}

function CompoundSearchFilterInputField({
    selectedEntity,
    selectedAttribute,
    value,
    searchFilter,
    onSearch,
    onChange,
    config,
}: CompoundSearchFilterInputFieldProps) {
    if (!selectedEntity || !selectedAttribute) {
        return null;
    }

    const entityObject = config[selectedEntity];
    const attributeObject: SearchFilterAttribute = entityObject?.attributes?.[selectedAttribute];

    if (!attributeObject) {
        return null;
    }

    if (attributeObject.inputType === 'text') {
        const textLabel = `Filter results by ${attributeObject.filterChipLabel.toLowerCase()}`;
        return (
            <SearchInput
                aria-label={textLabel}
                placeholder={textLabel}
                value={ensureString(value)}
                onChange={(_event, _value) => onChange(_value)}
                onSearch={(_event, _value) => {
                    onSearch({
                        action: 'ADD',
                        category: attributeObject.searchTerm,
                        value: _value,
                    });
                    onChange('');
                }}
                onClear={() => onChange('')}
                submitSearchButtonLabel="Apply text input to search"
            />
        );
    }
    if (attributeObject.inputType === 'date-picker') {
        return (
            <DatePicker
                aria-label="Filter by date"
                buttonAriaLabel="Filter by date toggle"
                value={ensureString(value)}
                onChange={(_event, _value) => {
                    const formattedValue = _value ? getDate(_value) : '';
                    onChange(_value);
                    onSearch({
                        action: 'ADD',
                        category: attributeObject.searchTerm,
                        value: formattedValue,
                    });
                }}
                dateFormat={getDate}
                placeholder={dateFormat}
            />
        );
    }
    if (attributeObject.inputType === 'condition-number') {
        return (
            <ConditionNumber
                value={ensureConditionNumber(value)}
                onChange={(newValue) => {
                    onChange(newValue);
                }}
                onSearch={(newValue) => {
                    const { condition, number } = newValue;
                    onChange(newValue);
                    onSearch({
                        action: 'ADD',
                        category: attributeObject.searchTerm,
                        value: `${conditionMap[condition]}${number}`,
                    });
                }}
            />
        );
    }
    if (
        entityObject &&
        entityObject.searchCategory &&
        attributeObject.inputType === 'autocomplete'
    ) {
        const { searchCategory } = entityObject;
        const { searchTerm, filterChipLabel } = attributeObject;
        const textLabel = `Filter results by ${filterChipLabel.toLowerCase()}`;
        return (
            <SearchFilterAutocomplete
                searchCategory={searchCategory}
                searchTerm={searchTerm}
                value={ensureString(value)}
                onChange={(newValue) => {
                    onChange(newValue);
                }}
                onSearch={(newValue) => {
                    onSearch({
                        action: 'ADD',
                        category: attributeObject.searchTerm,
                        value: newValue,
                    });
                    onChange('');
                }}
                textLabel={textLabel}
            />
        );
    }
    if (isSelectType(attributeObject)) {
        const attributeLabel = attributeObject.displayName.toLowerCase();
        const selectOptions = attributeObject.inputProps.options;
        const { searchTerm } = attributeObject;
        const selection = ensureStringArray(searchFilter?.[searchTerm]);

        return (
            <CheckboxSelect
                selection={selection}
                onChange={(checked, _value) => {
                    onChange(value);
                    onSearch({
                        action: checked ? 'ADD' : 'REMOVE',
                        category: attributeObject.searchTerm,
                        value: _value,
                    });
                }}
                ariaLabelMenu={`Filter by ${attributeLabel} select menu`}
                toggleLabel={`Filter by ${attributeLabel}`}
            >
                {selectOptions.length !== 0 ? (
                    selectOptions.map((option) => {
                        return (
                            <SelectOption
                                hasCheckbox
                                value={option.value}
                                isSelected={selection.includes(option.value)}
                            >
                                {option.label}
                            </SelectOption>
                        );
                    })
                ) : (
                    <SelectOption isDisabled>No options available</SelectOption>
                )}
            </CheckboxSelect>
        );
    }
    return <div />;
}

export default CompoundSearchFilterInputField;

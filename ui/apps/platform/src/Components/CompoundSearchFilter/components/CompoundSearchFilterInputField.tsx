import React from 'react';
import { Button, DatePicker, Flex, SearchInput, SelectOption } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { SearchFilter } from 'types/search';
import { getDate } from 'utils/dateUtils';
import { SelectedEntity } from './EntitySelector';
import { SelectedAttribute } from './AttributeSelector';
import {
    OnSearchPayload,
    PartialCompoundSearchFilterConfig,
    SearchFilterAttribute,
} from '../types';
import {
    conditionMap,
    ensureConditionNumber,
    ensureString,
    ensureStringArray,
    isSelectType,
} from '../utils/utils';

import CheckboxSelect from './CheckboxSelect';
import ConditionNumber from './ConditionNumber';
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
    additionalContextFilter?: SearchFilter;
    onSearch: ({ action, category, value }: OnSearchPayload) => void;
    onChange: InputFieldOnChange;
    config: PartialCompoundSearchFilterConfig;
};

function dateParse(date: string): Date {
    const split = date.split('/');
    if (split.length !== 3) {
        return new Date('Invalid Date');
    }
    const month = split[0];
    const day = split[1];
    const year = split[2];
    if (month.length !== 2 || day.length !== 2 || year.length !== 4) {
        return new Date('Invalid Date');
    }
    return new Date(
        `${year.padStart(4, '0')}-${month.padStart(2, '0')}-${day.padStart(2, '0')}T00:00:00`
    );
}

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

    const entityObject = config[selectedEntity];
    const attributeObject: SearchFilterAttribute = entityObject?.attributes?.[selectedAttribute];

    if (!attributeObject) {
        return null;
    }

    if (attributeObject.inputType === 'text') {
        const textLabel = `Filter results by ${attributeObject.filterChipLabel}`;
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
        const dateValue = ensureString(value);

        return (
            <Flex spaceItems={{ default: 'spaceItemsNone' }}>
                <DatePicker
                    aria-label="Filter by date"
                    buttonAriaLabel="Filter by date toggle"
                    value={dateValue}
                    onChange={(_event, _value) => {
                        onChange(_value);
                    }}
                    dateFormat={getDate}
                    dateParse={dateParse}
                    placeholder="MM/DD/YYYY"
                />
                <Button
                    variant="control"
                    aria-label="Apply date input to search"
                    onClick={() => {
                        const date = dateParse(dateValue);
                        if (!Number.isNaN(date.getTime())) {
                            onSearch({
                                action: 'ADD',
                                category: attributeObject.searchTerm,
                                value: dateValue,
                            });
                            onChange('');
                        }
                    }}
                >
                    <ArrowRightIcon />
                </Button>
            </Flex>
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
        const textLabel = `Filter results by ${filterChipLabel}`;
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
                searchFilter={searchFilter}
                additionalContextFilter={additionalContextFilter}
            />
        );
    }
    if (isSelectType(attributeObject)) {
        const attributeLabel = attributeObject.displayName;
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

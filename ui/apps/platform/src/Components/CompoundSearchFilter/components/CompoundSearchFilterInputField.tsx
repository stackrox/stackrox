import React from 'react';

import { SearchInput, SelectOption } from '@patternfly/react-core';
import { SelectedEntity } from './EntitySelector';
import { SelectedAttribute } from './AttributeSelector';
import {
    CompoundSearchFilterConfig,
    SearchFilterAttribute,
    SelectSearchFilterAttribute,
} from '../types';

import CheckboxSelect from './CheckboxSelect';

export type InputFieldValue = string | number | undefined | string[];
export type InputFieldOnChange = (value: InputFieldValue) => void;

export type CompoundSearchFilterInputFieldProps = {
    selectedEntity: SelectedEntity;
    selectedAttribute: SelectedAttribute;
    value: InputFieldValue;
    onSearch: InputFieldOnChange;
    onChange: InputFieldOnChange;
    config: Partial<CompoundSearchFilterConfig>;
};

function isSelectType(
    attributeObject: SearchFilterAttribute
): attributeObject is SelectSearchFilterAttribute {
    return attributeObject.inputType === 'select';
}

function ensureStringArray(value: InputFieldValue): string[] {
    if (value === undefined || typeof value === 'number' || typeof value === 'string') {
        return [];
    }
    return value;
}

function CompoundSearchFilterInputField({
    selectedEntity,
    selectedAttribute,
    value,
    onSearch,
    onChange,
    config,
}: CompoundSearchFilterInputFieldProps) {
    if (!selectedEntity || !selectedAttribute) {
        return null;
    }

    const attributeObject: SearchFilterAttribute =
        config[selectedEntity]?.attributes[selectedAttribute];

    if (!attributeObject) {
        return null;
    }

    if (attributeObject.inputType === 'text') {
        const textLabel = `Filter results by ${attributeObject.filterChipLabel.toLowerCase()}`;
        return (
            <SearchInput
                aria-label={textLabel}
                placeholder={textLabel}
                value={value as string}
                onChange={(_event, _value) => onChange(_value)}
                onSearch={(_event, _value) => onSearch(_value)}
                onClear={() => onChange('')}
            />
        );
    }
    if (isSelectType(attributeObject)) {
        const attributeLabel = attributeObject.displayName.toLowerCase();
        const selection = ensureStringArray(value);
        const selectOptions = attributeObject.inputProps.options;

        return (
            <CheckboxSelect
                selection={selection}
                onChange={(value) => {
                    onChange(value);
                    onSearch(value);
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

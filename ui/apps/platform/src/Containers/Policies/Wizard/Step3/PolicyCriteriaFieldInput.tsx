import React from 'react';
import { useField } from 'formik';
import {
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
    FormGroup,
    Select,
    SelectOption,
    SelectVariant,
} from '@patternfly/react-core';

import { Descriptor } from './policyCriteriaDescriptors';
import PolicyCriteriaFieldSubInput from './PolicyCriteriaFieldSubInput';
import TableModalFieldInput from './TableModalFieldInput';

type PolicyCriteriaFieldInputProps = {
    descriptor: Descriptor;
    readOnly?: boolean;
    name: string;
};

function PolicyCriteriaFieldInput({
    descriptor,
    readOnly = false,
    name,
}: PolicyCriteriaFieldInputProps): React.ReactElement {
    const [field, , helper] = useField(name);
    const [isSelectOpen, setIsSelectOpen] = React.useState(false);
    const { value } = field;
    const { setValue } = helper;

    function handleChangeValue(val) {
        setValue({ value: val });
    }

    function handleChangeSelectedValue(selectedVal) {
        return () => handleChangeValue(selectedVal);
    }

    function handleChangeSelect(e, val) {
        setIsSelectOpen(false);
        handleChangeValue(val);
    }

    function handleChangeSelectMultiple(e, selection) {
        if (value.value?.includes(selection)) {
            handleChangeValue(value.value.filter((item) => item !== selection));
        } else {
            handleChangeValue([...value.value, selection]);
        }
        setIsSelectOpen(false);
    }

    function handleOnToggleSelect() {
        setIsSelectOpen(!isSelectOpen);
    }

    /* eslint-disable default-case */
    switch (descriptor.type) {
        case 'text':
            return (
                <TextInput
                    value={value.value}
                    type="text"
                    id={name}
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                    data-testid="policy-criteria-value-text-input"
                    placeholder={descriptor.placeholder || ''}
                />
            );
        case 'radioGroup': {
            const booleanValue = value.value === true || value.value === 'true';
            return (
                <ToggleGroup data-testid="policy-criteria-value-radio-group">
                    {descriptor.radioButtons?.map(({ text, value: radioValue }) => (
                        <ToggleGroupItem
                            key={text}
                            text={text}
                            buttonId={text}
                            isDisabled={readOnly}
                            isSelected={booleanValue === radioValue}
                            onChange={handleChangeSelectedValue(radioValue)}
                            data-testid="policy-criteria-value-radio-group-item"
                        />
                    ))}
                </ToggleGroup>
            );
        }
        case 'radioGroupString': {
            return (
                <ToggleGroup data-testid="policy-criteria-value-radio-group-string">
                    {descriptor.radioButtons?.map(({ text, value: radioValue }) => (
                        <ToggleGroupItem
                            key={text}
                            text={text}
                            buttonId={text}
                            isDisabled={readOnly}
                            isSelected={value.value === radioValue}
                            onChange={handleChangeSelectedValue(radioValue)}
                            data-testid="policy-criteria-value-radio-group-string-item"
                        />
                    ))}
                </ToggleGroup>
            );
        }
        case 'number':
            return (
                <TextInput
                    value={value.value}
                    type="number"
                    id={name}
                    isDisabled={readOnly}
                    onChange={handleChangeValue}
                    data-testid="policy-criteria-value-number-input"
                />
            );
        case 'select':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-u-flex-1"
                    data-testid="policy-criteria-value-select"
                >
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelect}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value.value}
                        placeholderText={descriptor.placeholder || 'Select an option'}
                        menuAppendTo={() => document.body}
                    >
                        {descriptor?.options?.map((option) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                data-testid="policy-criteria-value-select-option"
                            >
                                {option.label}
                            </SelectOption>
                        ))}
                    </Select>
                </FormGroup>
            );
        case 'multiselect':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-u-flex-1"
                    data-testid="policy-criteria-value-multiselect"
                >
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelectMultiple}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value.value === '' ? [] : value.value}
                        onClear={handleChangeSelectedValue([])}
                        placeholderText={descriptor.placeholder || 'Select one or more options'}
                        variant={SelectVariant.typeaheadMulti}
                        menuAppendTo={() => document.body}
                    >
                        {descriptor.options?.map((option) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                data-testid="policy-criteria-value-multiselect-option"
                            >
                                {option.label}
                            </SelectOption>
                        ))}
                    </Select>
                </FormGroup>
            );
        case 'group': {
            /* eslint-disable react/no-array-index-key */
            return (
                <>
                    {descriptor.subComponents?.map((subComponent, index) => (
                        <PolicyCriteriaFieldSubInput
                            key={index}
                            subComponent={subComponent}
                            readOnly={readOnly}
                            name={`${name}.${subComponent.subpath}`}
                        />
                    ))}
                </>
            );
            /* eslint-enable react/no-array-index-key */
        }
        case 'tableModal': {
            return (
                <TableModalFieldInput
                    setValue={setValue}
                    value={value}
                    readOnly={readOnly}
                    tableType={descriptor.tableType}
                />
            );
        }
    }
    /* eslint-enable default-case */
}

export default PolicyCriteriaFieldInput;

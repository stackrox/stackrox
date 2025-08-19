import React from 'react';
import { useField } from 'formik';
import {
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
    FormGroup,
    Select,
    SelectOption,
    SelectList,
    MenuToggle,
    MenuToggleElement,
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
    const [isMultiSelectOpen, setIsMultiSelectOpen] = React.useState(false);
    const { value } = field;
    const { setValue } = helper;

    function handleChangeValue(val: string | string[] | boolean | number) {
        setValue({ value: val });
    }

    function handleChangeSelectedValue(selectedVal: string | string[] | boolean | number) {
        return () => handleChangeValue(selectedVal);
    }

    function handleChangeSelect(_e?: React.MouseEvent, val?: string | number) {
        setIsSelectOpen(false);
        handleChangeValue(val as string);
    }

    function handleChangeSelectMultiple(_e?: React.MouseEvent, selection?: string | number) {
        const selectionStr = selection as string;
        if (value.value?.includes && value.value.includes(selectionStr)) {
            handleChangeValue(
                (value.value as string[]).filter((item: string) => item !== selectionStr)
            );
        } else {
            handleChangeValue([...((value.value as string[]) || []), selectionStr]);
        }
    }

    function handleOnToggleSelect() {
        setIsSelectOpen(!isSelectOpen);
    }

    function handleOnToggleMultiSelect() {
        setIsMultiSelectOpen(!isMultiSelectOpen);
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
                    onChange={(_event, val) => handleChangeValue(val)}
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
                    onChange={(_event, val) => handleChangeValue(val)}
                    data-testid="policy-criteria-value-number-input"
                />
            );
        case 'select':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-v5-u-flex-1"
                    data-testid="policy-criteria-value-select"
                >
                    <Select
                        isOpen={isSelectOpen}
                        selected={value.value}
                        onSelect={handleChangeSelect}
                        onOpenChange={(nextOpen: boolean) => setIsSelectOpen(nextOpen)}
                        toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                            <MenuToggle
                                ref={toggleRef}
                                onClick={handleOnToggleSelect}
                                isExpanded={isSelectOpen}
                                isDisabled={readOnly}
                                data-testid="policy-criteria-value-select-toggle"
                            >
                                {value.value || descriptor.placeholder || 'Select an option'}
                            </MenuToggle>
                        )}
                        shouldFocusToggleOnSelect
                    >
                        <SelectList>
                            {descriptor?.options?.map((option) => (
                                <SelectOption
                                    key={option.value}
                                    value={option.value}
                                    data-testid="policy-criteria-value-select-option"
                                >
                                    {option.label}
                                </SelectOption>
                            ))}
                        </SelectList>
                    </Select>
                </FormGroup>
            );
        case 'multiselect':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-v5-u-flex-1"
                    data-testid="policy-criteria-value-multiselect"
                >
                    <Select
                        isOpen={isMultiSelectOpen}
                        selected={value.value === '' ? [] : value.value}
                        onSelect={handleChangeSelectMultiple}
                        onOpenChange={(nextOpen: boolean) => setIsMultiSelectOpen(nextOpen)}
                        toggle={(toggleRef: React.Ref<MenuToggleElement>) => {
                            const selections = value.value === '' ? [] : value.value;
                            const toggleText =
                                selections.length > 0
                                    ? `${selections.length} item${selections.length !== 1 ? 's' : ''} selected`
                                    : descriptor.placeholder || 'Select one or more options';
                            return (
                                <MenuToggle
                                    ref={toggleRef}
                                    onClick={handleOnToggleMultiSelect}
                                    isExpanded={isMultiSelectOpen}
                                    isDisabled={readOnly}
                                    data-testid="policy-criteria-value-multiselect-toggle"
                                >
                                    {toggleText}
                                </MenuToggle>
                            );
                        }}
                        shouldFocusToggleOnSelect
                    >
                        <SelectList>
                            {descriptor.options?.map((option) => {
                                const selections = value.value === '' ? [] : value.value;
                                const isSelected = selections.includes(option.value);
                                return (
                                    <SelectOption
                                        key={option.value}
                                        value={option.value}
                                        isSelected={isSelected}
                                        hasCheckbox
                                        data-testid="policy-criteria-value-multiselect-option"
                                    >
                                        {option.label}
                                    </SelectOption>
                                );
                            })}
                        </SelectList>
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

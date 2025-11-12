import type { ReactElement } from 'react';
import { useField } from 'formik';
import {
    FormGroup,
    SelectOption,
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle/SelectSingle';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';

import type { Descriptor } from './policyCriteriaDescriptors';
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
}: PolicyCriteriaFieldInputProps): ReactElement {
    const [field, , helper] = useField(name);
    const { value } = field;
    const { setValue } = helper;

    function handleChangeValue(val: string | string[] | boolean | number) {
        setValue({ value: val });
    }

    function handleChangeSelectedValue(selectedVal: string | string[] | boolean | number) {
        return () => handleChangeValue(selectedVal);
    }

    function handleChangeSelect(_id: string, val: string) {
        handleChangeValue(val);
    }

    function handleChangeSelectMultiple(newSelections: string[]) {
        handleChangeValue(newSelections);
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
                    <SelectSingle
                        id={descriptor.name}
                        value={value.value || ''}
                        handleSelect={handleChangeSelect}
                        isDisabled={readOnly}
                        placeholderText={descriptor.placeholder || 'Select an option'}
                    >
                        {descriptor?.options?.map((option) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                data-testid="policy-criteria-value-select-option"
                            >
                                {option.label}
                            </SelectOption>
                        )) || []}
                    </SelectSingle>
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
                    <CheckboxSelect
                        selections={(value.value as string[]) || []}
                        onChange={handleChangeSelectMultiple}
                        isDisabled={readOnly}
                        placeholderText={descriptor.placeholder || 'Select one or more options'}
                        ariaLabel={descriptor.label || 'Checkbox select menu'}
                    >
                        {descriptor.options?.map((option) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                data-testid="policy-criteria-value-multiselect-option"
                            >
                                {option.label}
                            </SelectOption>
                        )) || []}
                    </CheckboxSelect>
                </FormGroup>
            );
        case 'group': {
            /* eslint-disable react/no-array-index-key */
            const hasRowType = descriptor.subComponents?.some((sub) => sub.type === 'row');
            const wrapper = hasRowType ? (
                <div className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-gap-md">
                    {descriptor.subComponents?.map((subComponent, index) => {
                        const subInputName =
                            subComponent.type === 'row' ? name : `${name}.${subComponent.subpath}`;
                        return (
                            <div
                                key={index}
                                className={`pf-v5-u-w-100 ${index > 0 ? 'pf-v5-u-mt-md' : ''}`}
                            >
                                <PolicyCriteriaFieldSubInput
                                    subComponent={subComponent}
                                    readOnly={readOnly}
                                    name={subInputName}
                                    isInRowLayout
                                />
                            </div>
                        );
                    })}
                </div>
            ) : (
                <>
                    {descriptor.subComponents?.map((subComponent, index) => (
                        <PolicyCriteriaFieldSubInput
                            key={index}
                            subComponent={subComponent}
                            readOnly={readOnly}
                            name={`${name}.${(subComponent as { subpath: string }).subpath}`}
                        />
                    ))}
                </>
            );
            return wrapper;
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

import React from 'react';
import { useField } from 'formik';
import { TextInput, NumberInput, FormGroup, Select, SelectOption } from '@patternfly/react-core';

import { SubComponent } from 'Containers/Policies/Wizard/Form/descriptors';

type PolicyCriteriaFieldSubInputProps = {
    subComponent: SubComponent;
    readOnly?: boolean;
    name: string;
};

function PolicyCriteriaFieldSubInput({
    subComponent,
    readOnly = false,
    name,
}: PolicyCriteriaFieldSubInputProps): React.ReactElement {
    const [field, , helper] = useField(name);
    const [isSelectOpen, setIsSelectOpen] = React.useState(false);
    const { value } = field;
    const { setValue } = helper;

    function handleChangeNumberValue(e) {
        const newValue = Number.isNaN(e.target.value) ? 0 : Number(e.target.value);
        const { max = 10, min = 0 } = subComponent;
        if (newValue > max) {
            setValue(max);
        } else if (newValue < min) {
            setValue(min);
        } else {
            setValue(newValue);
        }
    }

    function handleChangeSelect(e, val) {
        setIsSelectOpen(false);
        setValue(val);
    }

    function handleOnToggleSelect() {
        setIsSelectOpen(!isSelectOpen);
    }

    function handleOnMinus(step = 1) {
        return () => setValue((Number(value) - step).toFixed(1));
    }

    function handleOnPlus(step = 1) {
        return () => setValue((Number(value) + step).toFixed(1));
    }

    /* eslint-disable default-case */
    switch (subComponent.type) {
        case 'text':
            return (
                <FormGroup label={subComponent.label} fieldId={name} className="pf-u-flex-1">
                    <TextInput
                        value={value}
                        type="text"
                        isDisabled={readOnly}
                        placeholder={subComponent.placeholder}
                        onChange={(v) => setValue(v)}
                    />
                </FormGroup>
            );
        case 'number':
            return (
                <NumberInput
                    value={Number(value)}
                    isDisabled={readOnly}
                    onChange={handleChangeNumberValue}
                    min={subComponent.min}
                    max={subComponent.max}
                    onPlus={handleOnPlus(subComponent.step)}
                    onMinus={handleOnMinus(subComponent.step)}
                />
            );
        case 'select':
            return (
                <FormGroup label={subComponent.label} fieldId={name} className="pf-u-flex-1">
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelect}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value}
                        placeholderText={subComponent.placeholder || 'Select an option'}
                    >
                        {subComponent.options?.map((option) => (
                            <SelectOption key={option.value} value={option.value}>
                                {option.label}
                            </SelectOption>
                        ))}
                    </Select>
                </FormGroup>
            );
    }
    /* eslint-enable default-case */
}

export default PolicyCriteriaFieldSubInput;

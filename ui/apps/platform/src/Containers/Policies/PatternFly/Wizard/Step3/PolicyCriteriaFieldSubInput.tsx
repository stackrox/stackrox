import React from 'react';
import { useField } from 'formik';
import { TextInput, FormGroup, Select, SelectOption } from '@patternfly/react-core';

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

    function handleChangeNumberValue(val) {
        const newValue = Number.isNaN(val) ? 0 : Number(val);
        const { max, min = 0 } = subComponent;
        if (max && newValue > max) {
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

    /* eslint-disable default-case */
    switch (subComponent.type) {
        case 'text':
            return (
                <FormGroup label={subComponent.label} fieldId={name} className="pf-u-flex-1">
                    <TextInput
                        value={value}
                        type="text"
                        id={name}
                        isDisabled={readOnly}
                        placeholder={subComponent.placeholder}
                        onChange={(v) => setValue(v)}
                    />
                </FormGroup>
            );
        case 'number':
            return (
                <TextInput
                    value={Number(value)}
                    type="number"
                    id={name}
                    isDisabled={readOnly}
                    onChange={handleChangeNumberValue}
                    className="pf-u-w-25"
                />
            );
        case 'select':
            return (
                <FormGroup
                    label={subComponent.label}
                    fieldId={name}
                    className="pf-u-flex-1 pf-u-w-0"
                >
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

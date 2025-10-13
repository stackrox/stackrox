import React from 'react';
import { useField } from 'formik';
import { TextInput, FormGroup, SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { SubComponent } from './policyCriteriaDescriptors';

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
    const { value } = field;
    const { setValue } = helper;

    function handleSelectChange(name: string, value: string) {
        setValue(value);
    }

    /* eslint-disable default-case */
    switch (subComponent.type) {
        case 'text':
            return (
                <FormGroup label={subComponent.label} fieldId={name} className="pf-v5-u-flex-1">
                    <TextInput
                        value={value}
                        type="text"
                        id={name}
                        isDisabled={readOnly}
                        onChange={(_event, v) => setValue(v)}
                        data-testid="policy-criteria-value-text-input"
                    />
                </FormGroup>
            );
        case 'number':
            return (
                <TextInput
                    value={value}
                    type="number"
                    id={name}
                    isDisabled={readOnly}
                    onChange={(_event, v) => setValue(v)}
                    placeholder="(ex. 5)"
                    className="pf-v5-u-w-25"
                    data-testid="policy-criteria-value-number-input"
                />
            );
        case 'select':
            return (
                <FormGroup
                    label={subComponent.label}
                    fieldId={name}
                    className="pf-v5-u-flex-1 pf-v5-u-w-0"
                    data-testid="policy-criteria-value-select"
                >
                    <SelectSingle
                        id={name}
                        value={value || ''}
                        handleSelect={handleSelectChange}
                        isDisabled={readOnly}
                        placeholderText={subComponent.placeholder || 'Select an option'}
                        menuAppendTo={() => document.body}
                    >
                        {subComponent.options?.map((option) => (
                            <SelectOption
                                key={option.value}
                                value={option.value}
                                data-testid="policy-criteria-value-select-option"
                            >
                                {option.label}
                            </SelectOption>
                        ))}
                    </SelectSingle>
                </FormGroup>
            );
    }
    /* eslint-enable default-case */
}

export default PolicyCriteriaFieldSubInput;

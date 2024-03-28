import React from 'react';
import { useField } from 'formik';
import { TextInput, FormGroup } from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';

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
    const [isSelectOpen, setIsSelectOpen] = React.useState(false);
    const { value } = field;
    const { setValue } = helper;

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
                    <Select
                        onToggle={handleOnToggleSelect}
                        onSelect={handleChangeSelect}
                        isOpen={isSelectOpen}
                        isDisabled={readOnly}
                        selections={value}
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
                    </Select>
                </FormGroup>
            );
    }
    /* eslint-enable default-case */
}

export default PolicyCriteriaFieldSubInput;

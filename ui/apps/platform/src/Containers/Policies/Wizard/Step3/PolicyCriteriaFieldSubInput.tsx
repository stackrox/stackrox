import type { ReactElement } from 'react';
import { useField } from 'formik';
import { FormGroup, SelectOption, TextInput } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle/SelectSingle';
import type { SubComponent } from './policyCriteriaDescriptors';

type PolicyCriteriaFieldSubInputProps = {
    subComponent: SubComponent;
    readOnly?: boolean;
    name: string;
    isInRowLayout?: boolean;
};

function PolicyCriteriaFieldSubInput({
    subComponent,
    readOnly = false,
    name,
    isInRowLayout = false,
}: PolicyCriteriaFieldSubInputProps): ReactElement {
    const [field, , helper] = useField(name);
    const { value } = field;
    const { setValue } = helper;

    function handleSelectChange(_name: string, value: string) {
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
        case 'select': {
            const formGroupClass = isInRowLayout ? '' : 'pf-v5-u-flex-1 pf-v5-u-w-0';
            return (
                <FormGroup
                    label={subComponent.label}
                    fieldId={name}
                    className={formGroupClass}
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
        case 'row': {
            return (
                <div className="pf-v5-u-display-flex pf-v5-u-w-100">
                    {subComponent.children.map((child, index) => {
                        // Row children must have subpath (not nested rows)
                        const childSubpath = (child as { subpath: string }).subpath;
                        // Use subpath as key since it's stable and unique per row
                        return (
                            <div
                                key={childSubpath}
                                className={`pf-v5-u-flex pf-v5-u-flex-1 ${index > 0 ? 'pf-v5-u-ml-md' : ''}`}
                            >
                                <PolicyCriteriaFieldSubInput
                                    subComponent={child}
                                    readOnly={readOnly}
                                    name={`${name}.${childSubpath}`}
                                />
                            </div>
                        );
                    })}
                </div>
            );
        }
    }
    /* eslint-enable default-case */
}

export default PolicyCriteriaFieldSubInput;

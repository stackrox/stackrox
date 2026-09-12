import type { ReactElement } from 'react';
import { useField, useFormikContext } from 'formik';
import {
    Flex,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    SelectOption,
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle/SelectSingle';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import type { ClientPolicy } from 'types/policy.proto';

import type { Descriptor } from './policyCriteriaDescriptors';
import { auditLogAllowedVerbsByResource } from './policyCriteriaDescriptors';
import PolicyCriteriaFieldSubInput from './PolicyCriteriaFieldSubInput';
import TableModalFieldInput from './TableModalFieldInput';
import { getAvailableOptionsForField } from './policyCriteriaUtils';

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
    const { values } = useFormikContext<ClientPolicy>();

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
        case 'text': {
            // value.value is always a string for 'text' type descriptors
            const validationError = descriptor.validate?.(String(value.value));
            const showError = Boolean(validationError);
            const warningMessage = !showError ? descriptor.warn?.(String(value.value)) : undefined;
            const showWarning = Boolean(warningMessage);

            const feedbackVariant = showError ? 'error' : showWarning ? 'warning' : 'default';
            const feedbackMessage = validationError ?? warningMessage ?? descriptor.helperText;

            return (
                <Flex grow={{ default: 'grow' }}>
                    <TextInput
                        value={value.value}
                        type="text"
                        id={name}
                        isDisabled={readOnly}
                        onChange={(_event, val) => handleChangeValue(val)}
                        data-testid="policy-criteria-value-text-input"
                        placeholder={descriptor.placeholder || ''}
                        validated={feedbackVariant}
                    />
                    {feedbackMessage && (
                        <FormHelperText>
                            <HelperText isLiveRegion={showError || showWarning}>
                                <HelperTextItem variant={feedbackVariant}>
                                    {feedbackMessage}
                                </HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    )}
                </Flex>
            );
        }
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
        case 'select': {
            let filteredOptions = getAvailableOptionsForField(descriptor.options, name, values);

            // For audit log policies, filter verb options based on the selected
            // resources in the same section (and vice versa). Values within a
            // group are OR'd, so a verb is shown if it's allowed for ANY
            // selected resource.
            if (descriptor.name === 'Kubernetes API Verb' || descriptor.name === 'Kubernetes Resource') {
                const sectionMatch = name.match(/^policySections\[(\d+)\]/);
                if (sectionMatch) {
                    const sectionIndex = parseInt(sectionMatch[1], 10);
                    const section = values.policySections[sectionIndex];
                    if (section) {
                        if (descriptor.name === 'Kubernetes API Verb') {
                            const resourceGroup = section.policyGroups.find(
                                (g) => g.fieldName === 'Kubernetes Resource'
                            );
                            const selectedResources = (resourceGroup?.values ?? [])
                                .map((v) => (typeof v.value === 'string' ? v.value.toUpperCase() : ''))
                                .filter(Boolean);
                            if (selectedResources.length > 0) {
                                filteredOptions = filteredOptions.filter((opt) =>
                                    selectedResources.some((res) => {
                                        const allowed = auditLogAllowedVerbsByResource[res];
                                        return allowed?.includes(opt.value.toUpperCase()) ?? true;
                                    })
                                );
                            }
                        } else if (descriptor.name === 'Kubernetes Resource') {
                            const verbGroup = section.policyGroups.find(
                                (g) => g.fieldName === 'Kubernetes API Verb'
                            );
                            const selectedVerbs = (verbGroup?.values ?? [])
                                .map((v) => (typeof v.value === 'string' ? v.value.toUpperCase() : ''))
                                .filter(Boolean);
                            if (selectedVerbs.length > 0) {
                                filteredOptions = filteredOptions.filter((opt) => {
                                    const allowed = auditLogAllowedVerbsByResource[opt.value.toUpperCase()];
                                    return !allowed || selectedVerbs.some((v) => allowed.includes(v));
                                });
                            }
                        }
                    }
                }
            }

            const availableOptions = filteredOptions;

            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-v6-u-flex-1"
                    data-testid="policy-criteria-value-select"
                >
                    <SelectSingle
                        id={descriptor.name}
                        value={value.value || ''}
                        handleSelect={handleChangeSelect}
                        isDisabled={readOnly}
                        placeholderText={descriptor.placeholder || 'Select an option'}
                    >
                        {availableOptions.map((option) => (
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
        case 'multiselect':
            return (
                <FormGroup
                    label={descriptor.label}
                    fieldId={descriptor.name}
                    className="pf-v6-u-flex-1"
                    data-testid="policy-criteria-value-multiselect"
                >
                    <CheckboxSelect
                        selections={(value.value as string[]) ?? []}
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
                        )) ?? []}
                    </CheckboxSelect>
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

import React from 'react';
import {
    Button,
    FormGroup,
    SelectOption,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import cloneDeep from 'lodash/cloneDeep';

import { FormikErrors } from 'formik';
import useIndexKey from 'hooks/useIndexKey';
import {
    ByNameMatchType,
    ByNameResourceSelector,
    ScopedResourceSelector,
    SelectorEntityType,
} from '../types';
import { NameMatchTypeSelect } from './MatchTypeSelect';

function parseInlineRuleError(
    errors: ByNameSelectorProps['validationErrors'],
    valueIndex: number
): string | undefined {
    const valueErrors = errors?.rule?.values?.[valueIndex];
    if (typeof valueErrors === 'string') {
        return valueErrors;
    }
    return valueErrors?.value;
}

export type ByNameSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ByNameResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    placeholder: string;
    validationErrors: FormikErrors<ByNameResourceSelector> | undefined;
    isDisabled: boolean;
};

function ByNameSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
    placeholder,
    validationErrors,
    isDisabled,
}: ByNameSelectorProps) {
    const { keyFor, invalidateIndexKeys } = useIndexKey();
    const lowerCaseEntity = entityType.toLowerCase();

    function onAddValue() {
        const selector = cloneDeep(scopedResourceSelector);
        // Only add a new form row if there are no blank entries
        if (selector.rule.values.every(({ value }) => value)) {
            selector.rule.values.push({ value: '', matchType: 'EXACT' });
            handleChange(entityType, selector);
        }
    }

    function onChangeMatchType(resourceSelector: ByNameResourceSelector, valueIndex: number) {
        return (value: ByNameMatchType) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex].matchType = value;
            handleChange(entityType, newSelector);
        };
    }

    function onChangeValue(resourceSelector, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex].value = value;
            handleChange(entityType, newSelector);
        };
    }

    function onDeleteValue(valueIndex: number) {
        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rule.values.length > 1) {
            newSelector.rule.values.splice(valueIndex, 1);
            invalidateIndexKeys();
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the rule, so drop the selector
            handleChange(entityType, { type: 'All' });
        }
    }

    return (
        <>
            <div className="rule-selector-list">
                {scopedResourceSelector.rule.values.map(({ matchType, value }, index) => {
                    const inputId = `${entityType}-name-value-${index}`;
                    const inputAriaLabel = `Select value ${index + 1} of ${
                        scopedResourceSelector.rule.values.length
                    } for the ${lowerCaseEntity} name`;
                    const inputClassName = 'pf-u-flex-grow-1 pf-u-w-auto';
                    const inputOnChange = onChangeValue(scopedResourceSelector, index);
                    const errorMessage = parseInlineRuleError(validationErrors, index);
                    const inputValidated = errorMessage
                        ? ValidatedOptions.error
                        : ValidatedOptions.default;

                    return (
                        <div className="rule-selector-list-item" key={keyFor(index)}>
                            <div className="rule-selector-match-type-select">
                                <NameMatchTypeSelect
                                    selected={matchType}
                                    isDisabled={isDisabled}
                                    onChange={onChangeMatchType(scopedResourceSelector, index)}
                                >
                                    <SelectOption value="EXACT">An exact value of</SelectOption>
                                    <SelectOption value="REGEX">A regex value of</SelectOption>
                                </NameMatchTypeSelect>
                            </div>
                            <div className="rule-selector-name-value-input">
                                <FormGroup
                                    className="rule-selector-name-value-input"
                                    fieldId={inputId}
                                    helperTextInvalid={errorMessage}
                                    validated={inputValidated}
                                >
                                    <TextInput
                                        id={inputId}
                                        aria-label={inputAriaLabel}
                                        placeholder={
                                            matchType === 'REGEX' ? `^${placeholder}$` : placeholder
                                        }
                                        className={inputClassName}
                                        onChange={inputOnChange}
                                        validated={inputValidated}
                                        value={value}
                                        isDisabled={isDisabled}
                                    />
                                </FormGroup>
                            </div>
                            {!isDisabled && (
                                <Button
                                    className="rule-selector-delete-value-button"
                                    aria-label={`Delete ${value}`}
                                    variant="plain"
                                    onClick={() => onDeleteValue(index)}
                                >
                                    <TrashIcon
                                        className="pf-u-flex-shrink-1"
                                        style={{ cursor: 'pointer' }}
                                        color="var(--pf-global--Color--dark-200)"
                                    />
                                </Button>
                            )}
                        </div>
                    );
                })}
            </div>
            {!isDisabled && (
                <Button
                    aria-label={`Add ${lowerCaseEntity} name value`}
                    className="rule-selector-add-value-button"
                    variant="link"
                    onClick={onAddValue}
                >
                    OR...
                </Button>
            )}
        </>
    );
}

export default ByNameSelector;

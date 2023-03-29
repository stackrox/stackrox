import React from 'react';
import {
    Button,
    Divider,
    FormGroup,
    Label,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { FormikErrors } from 'formik';
import cloneDeep from 'lodash/cloneDeep';

import useIndexKey from 'hooks/useIndexKey';
import { ByLabelResourceSelector, ScopedResourceSelector, SelectorEntityType } from '../types';

function parseInlineRuleError(
    errors: ByLabelSelectorProps['validationErrors'],
    ruleIndex: number,
    valueIndex: number
): string | undefined {
    const ruleErrors = errors?.rules?.[ruleIndex];
    if (typeof ruleErrors === 'string') {
        return ruleErrors;
    }
    const valueErrors = ruleErrors?.values?.[valueIndex];
    if (typeof valueErrors === 'string') {
        return valueErrors;
    }
    return valueErrors?.value;
}

export type ByLabelSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ByLabelResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    placeholder: string;
    validationErrors: FormikErrors<ByLabelResourceSelector> | undefined;
    isDisabled: boolean;
};

function ByLabelSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
    placeholder,
    validationErrors,
    isDisabled,
}: ByLabelSelectorProps) {
    const { keyFor, invalidateIndexKeys } = useIndexKey();
    const lowerCaseEntity = entityType.toLowerCase();

    function onChangeLabelValue(
        resourceSelector: ByLabelResourceSelector,
        ruleIndex: number,
        valueIndex: number,
        value: string
    ) {
        const newSelector = cloneDeep(resourceSelector);
        newSelector.rules[ruleIndex].values[valueIndex].value = value;
        handleChange(entityType, newSelector);
    }

    function onAddLabelRule() {
        const selector = cloneDeep(scopedResourceSelector);

        // Only add a new form row if there are no blank entries
        if (selector.rules.every(({ values }) => values.every(({ value }) => value))) {
            selector.rules.push({
                operator: 'OR',
                values: [{ value: '', matchType: 'EXACT' }],
            });
            handleChange(entityType, selector);
        }
    }

    function onAddLabelValue(ruleIndex: number) {
        const selector = cloneDeep(scopedResourceSelector);
        const rule = selector.rules[ruleIndex];

        // Only add a new form row if there are no blank entries
        if (rule.values.every(({ value }) => value)) {
            rule.values.push({ value: '', matchType: 'EXACT' });
            handleChange(entityType, selector);
        }
    }

    function onDeleteValue(ruleIndex: number, valueIndex: number) {
        if (!scopedResourceSelector.rules[ruleIndex]) {
            return;
        }

        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rules[ruleIndex].values.length > 1) {
            newSelector.rules[ruleIndex].values.splice(valueIndex, 1);
            invalidateIndexKeys();
            handleChange(entityType, newSelector);
        } else if (newSelector.rules.length > 1) {
            // This was the last value, so drop the rule
            newSelector.rules.splice(ruleIndex, 1);
            invalidateIndexKeys();
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the last rule, so drop the selector
            handleChange(entityType, { type: 'All' });
        }
    }

    return (
        <>
            {scopedResourceSelector.rules.map((rule, ruleIndex) => {
                return (
                    <div key={keyFor(ruleIndex)}>
                        {ruleIndex > 0 && (
                            <div className="rule-selector-label-rule-separator">
                                <Label variant="outline" isCompact>
                                    and
                                </Label>
                            </div>
                        )}

                        <div className="rule-selector-list">
                            {rule.values.map(({ value }, valueIndex) => {
                                const errorMessage = parseInlineRuleError(
                                    validationErrors,
                                    ruleIndex,
                                    valueIndex
                                );
                                const inputValidated = errorMessage
                                    ? ValidatedOptions.error
                                    : ValidatedOptions.default;
                                const inputId = `${entityType}-label-value-${ruleIndex}-${valueIndex}`;
                                const ariaLabel = `Select label value ${valueIndex + 1} of ${
                                    rule.values.length
                                } for ${lowerCaseEntity} rule ${ruleIndex + 1} of ${
                                    scopedResourceSelector.rules.length
                                }`;
                                return (
                                    <div
                                        className="rule-selector-list-item"
                                        key={keyFor(valueIndex)}
                                    >
                                        <FormGroup
                                            className="rule-selector-name-value-input"
                                            fieldId={inputId}
                                            helperTextInvalid={errorMessage}
                                            validated={inputValidated}
                                        >
                                            <TextInput
                                                id={inputId}
                                                aria-label={ariaLabel}
                                                className="pf-u-flex-grow-1 pf-u-w-auto"
                                                onChange={(val) =>
                                                    onChangeLabelValue(
                                                        scopedResourceSelector,
                                                        ruleIndex,
                                                        valueIndex,
                                                        val
                                                    )
                                                }
                                                placeholder={placeholder}
                                                validated={inputValidated}
                                                value={value}
                                                isDisabled={isDisabled}
                                            />
                                        </FormGroup>
                                        {!isDisabled && (
                                            <Button
                                                className="rule-selector-delete-value-button"
                                                aria-label={`Delete ${value}`}
                                                variant="plain"
                                                onClick={() => onDeleteValue(ruleIndex, valueIndex)}
                                            >
                                                <TrashIcon
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
                                aria-label={`Add ${lowerCaseEntity} label value for rule ${
                                    ruleIndex + 1
                                }`}
                                className="rule-selector-add-value-button"
                                variant="link"
                                onClick={() => onAddLabelValue(ruleIndex)}
                            >
                                OR...
                            </Button>
                        )}
                    </div>
                );
            })}
            {!isDisabled && (
                <div className="pf-u-pt-md">
                    <Divider component="div" className="pf-u-pb-md" />
                    <Button
                        aria-label={`Add ${lowerCaseEntity} label rule`}
                        className="pf-u-p-0"
                        variant="link"
                        onClick={onAddLabelRule}
                    >
                        Add label section (AND)
                    </Button>
                </div>
            )}
        </>
    );
}

export default ByLabelSelector;

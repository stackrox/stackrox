import React from 'react';
import {
    Flex,
    Label,
    FormGroup,
    FlexItem,
    Button,
    Divider,
    ValidatedOptions,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { FormikErrors } from 'formik';
import cloneDeep from 'lodash/cloneDeep';

import useIndexKey from 'hooks/useIndexKey';
import { SelectorEntityType, ScopedResourceSelector, ByLabelResourceSelector } from '../types';
import { AutoCompleteSelect } from './AutoCompleteSelect';

export type ByLabelSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ByLabelResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    validationErrors: FormikErrors<ByLabelResourceSelector> | undefined;
    isDisabled: boolean;
};

function ByLabelSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
    validationErrors,
    isDisabled,
}: ByLabelSelectorProps) {
    const { keyFor, invalidateIndexKeys } = useIndexKey();
    const lowerCaseEntity = entityType.toLowerCase();
    function onChangeLabelKey(resourceSelector: ByLabelResourceSelector, ruleIndex, value) {
        const newSelector = cloneDeep(resourceSelector);
        newSelector.rules[ruleIndex].key = value;
        handleChange(entityType, newSelector);
    }

    function onChangeLabelValue(resourceSelector, ruleIndex, valueIndex, value) {
        const newSelector = cloneDeep(resourceSelector);
        newSelector.rules[ruleIndex].values[valueIndex] = value;
        handleChange(entityType, newSelector);
    }

    function onAddLabelRule() {
        const selector = cloneDeep(scopedResourceSelector);

        // Only add a new form row if there are no blank entries
        if (selector.rules.every(({ key, values }) => key && values.every((value) => value))) {
            selector.rules.push({ operator: 'OR', key: '', values: [''] });
            handleChange(entityType, selector);
        }
    }

    function onAddLabelValue(ruleIndex: number) {
        const selector = cloneDeep(scopedResourceSelector);
        const rule = selector.rules[ruleIndex];

        // Only add a new form row if there are no blank entries
        if (rule.values.every((value) => value)) {
            rule.values.push('');
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
                const keyValidationError = validationErrors?.rules?.[ruleIndex];
                const keyValidation =
                    typeof keyValidationError === 'string' || keyValidationError?.key
                        ? ValidatedOptions.error
                        : ValidatedOptions.default;
                return (
                    <div key={keyFor(ruleIndex)}>
                        {ruleIndex > 0 && (
                            <Flex
                                className="pf-u-pt-md pf-u-pb-xl"
                                spaceItems={{ default: 'spaceItemsNone' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                            >
                                <Label variant="outline" isCompact>
                                    and
                                </Label>
                                <span
                                    style={{
                                        borderBottom:
                                            '2px dashed var(--pf-global--Color--light-300)',
                                        flex: '1 1 0',
                                    }}
                                />
                            </Flex>
                        )}

                        <Flex>
                            <Flex className="pf-u-flex-grow-1 pf-u-mb-md">
                                <FormGroup
                                    fieldId={`${entityType}-label-key-${ruleIndex}`}
                                    className="pf-u-flex-grow-1"
                                    label={ruleIndex === 0 ? 'Label key' : ''}
                                    isRequired
                                >
                                    <AutoCompleteSelect
                                        id={`${entityType}-label-key-${ruleIndex}`}
                                        typeAheadAriaLabel={`Select label key for ${lowerCaseEntity} rule ${
                                            ruleIndex + 1
                                        } of ${scopedResourceSelector.rules.length}`}
                                        selectedOption={rule.key}
                                        onChange={(fieldValue: string) =>
                                            onChangeLabelKey(
                                                scopedResourceSelector,
                                                ruleIndex,
                                                fieldValue
                                            )
                                        }
                                        validated={keyValidation}
                                        isDisabled={isDisabled}
                                    />
                                </FormGroup>
                                <FlexItem
                                    className="pf-u-pb-xs"
                                    alignSelf={{ default: 'alignSelfFlexEnd' }}
                                >
                                    =
                                </FlexItem>
                            </Flex>
                            <FormGroup
                                fieldId={`${entityType}-label-value-${ruleIndex}`}
                                className="pf-u-flex-grow-1"
                                label={ruleIndex === 0 ? 'Label value(s)' : ''}
                                isRequired
                            >
                                <Flex
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    direction={{ default: 'column' }}
                                >
                                    {rule.values.map((value, valueIndex) => {
                                        const valueValidationError =
                                            validationErrors?.rules?.[ruleIndex];
                                        const valueValidation =
                                            typeof valueValidationError === 'string' ||
                                            valueValidationError?.values?.[valueIndex]
                                                ? ValidatedOptions.error
                                                : ValidatedOptions.default;
                                        return (
                                            <Flex key={keyFor(valueIndex)}>
                                                <AutoCompleteSelect
                                                    id={`${entityType}-label-value-${ruleIndex}-${valueIndex}`}
                                                    typeAheadAriaLabel={`Select label value ${
                                                        valueIndex + 1
                                                    } of ${
                                                        rule.values.length
                                                    } for ${lowerCaseEntity} rule ${
                                                        ruleIndex + 1
                                                    } of ${scopedResourceSelector.rules.length}`}
                                                    className="pf-u-flex-grow-1 pf-u-w-auto"
                                                    selectedOption={value}
                                                    onChange={(fieldValue: string) =>
                                                        onChangeLabelValue(
                                                            scopedResourceSelector,
                                                            ruleIndex,
                                                            valueIndex,
                                                            fieldValue
                                                        )
                                                    }
                                                    validated={valueValidation}
                                                    isDisabled={isDisabled}
                                                />
                                                {!isDisabled && (
                                                    <Button
                                                        aria-label={`Delete ${value}`}
                                                        variant="plain"
                                                        onClick={() =>
                                                            onDeleteValue(ruleIndex, valueIndex)
                                                        }
                                                    >
                                                        <TrashIcon
                                                            style={{ cursor: 'pointer' }}
                                                            color="var(--pf-global--Color--dark-200)"
                                                        />
                                                    </Button>
                                                )}
                                            </Flex>
                                        );
                                    })}
                                </Flex>
                                {!isDisabled && (
                                    <Button
                                        aria-label={`Add ${lowerCaseEntity} label value for rule ${
                                            ruleIndex + 1
                                        }`}
                                        className="pf-u-pl-0 pf-u-pt-md"
                                        variant="link"
                                        onClick={() => onAddLabelValue(ruleIndex)}
                                    >
                                        Add value
                                    </Button>
                                )}
                            </FormGroup>
                        </Flex>
                    </div>
                );
            })}
            {!isDisabled && (
                <>
                    <Divider component="div" className="pf-u-pt-lg" />
                    <Button
                        aria-label={`Add ${lowerCaseEntity} label rule`}
                        className="pf-u-pl-0 pf-u-pt-md"
                        variant="link"
                        onClick={onAddLabelRule}
                    >
                        Add label rule
                    </Button>
                </>
            )}
        </>
    );
}

export default ByLabelSelector;

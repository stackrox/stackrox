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
};

function ByLabelSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
    validationErrors,
}: ByLabelSelectorProps) {
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
            handleChange(entityType, newSelector);
        } else if (newSelector.rules.length > 1) {
            // This was the last value, so drop the rule
            newSelector.rules.splice(ruleIndex, 1);
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the last rule, so drop the selector
            handleChange(entityType, {});
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
                    <div key={rule.key}>
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
                                    className="pf-u-flex-grow-1"
                                    label={ruleIndex === 0 ? 'Label key' : ''}
                                    isRequired
                                >
                                    <AutoCompleteSelect
                                        selectedOption={rule.key}
                                        onChange={(fieldValue: string) =>
                                            onChangeLabelKey(
                                                scopedResourceSelector,
                                                ruleIndex,
                                                fieldValue
                                            )
                                        }
                                        validated={keyValidation}
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
                                            <Flex key={value}>
                                                <AutoCompleteSelect
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
                                                />
                                                <Button
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
                                            </Flex>
                                        );
                                    })}
                                </Flex>
                                <Button
                                    className="pf-u-pl-0 pf-u-pt-md"
                                    variant="link"
                                    onClick={() => onAddLabelValue(ruleIndex)}
                                >
                                    Add value
                                </Button>
                            </FormGroup>
                        </Flex>
                    </div>
                );
            })}
            <Divider component="div" className="pf-u-pt-lg" />
            <Button className="pf-u-pl-0 pf-u-pt-md" variant="link" onClick={onAddLabelRule}>
                Add label rule
            </Button>
        </>
    );
}

export default ByLabelSelector;

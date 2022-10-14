import React from 'react';
import { Flex, Label, FormGroup, FlexItem, Button, Divider } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import cloneDeep from 'lodash/cloneDeep';

import { SelectorEntityType, ScopedResourceSelector } from '../types';
import { AutoCompleteSelect } from './AutoCompleteSelect';

export type ByLabelSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ScopedResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function ByLabelSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
}: ByLabelSelectorProps) {
    // TODO Better validation for regex (disallow '=' in user entered values ??)
    function onChangeLabelKey(resourceSelector, ruleIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            const currentValues = newSelector.rules[ruleIndex].values;
            newSelector.rules[ruleIndex].values = currentValues.map((label) => ({
                value: label.value.replace(/.*=/, `${value}=`),
            }));
            handleChange(entityType, newSelector);
        };
    }

    function onChangeLabelValue(resourceSelector, ruleIndex, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            const targetValue = newSelector.rules[ruleIndex].values[valueIndex].value;
            newSelector.rules[ruleIndex].values[valueIndex] = {
                value: targetValue.replace(/=.*/, `=${value}`),
            };
            handleChange(entityType, newSelector);
        };
    }

    function onAddLabelRule() {
        const selector = cloneDeep(scopedResourceSelector);
        const lastRule = selector?.rules[selector?.rules.length - 1];

        // Only add a new form row if there are no blank entries
        if (!lastRule || lastRule.values.every(({ value }) => value === '=')) {
            return;
        }

        selector.rules.push({ operator: 'OR', values: [{ value: '=' }] });
        handleChange(entityType, selector);
    }

    function onAddLabelValue(ruleIndex: number, labelKey: string) {
        const selector = cloneDeep(scopedResourceSelector);
        const rule = selector?.rules[ruleIndex];
        const keyPrefix = `${labelKey}=`;

        // Only add a new form row if there are no blank entries
        if (!rule || !rule.values.every(({ value }) => value.replace(keyPrefix, ''))) {
            return;
        }

        rule.values.push({ value: keyPrefix });
        handleChange(entityType, selector);
    }

    function onDeleteValue(ruleIndex: number, valueIndex: number) {
        if (!scopedResourceSelector || !scopedResourceSelector.rules[ruleIndex]) {
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
            handleChange(entityType, null);
        }
    }
    return (
        <>
            {scopedResourceSelector.rules.map((rule, ruleIndex) => {
                const labelKey = rule.values[0]?.value?.split('=')[0] ?? '';
                return (
                    <div key={labelKey}>
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
                            <Flex className="pf-u-flex-grow-1">
                                <FormGroup
                                    className="pf-u-flex-grow-1"
                                    label={ruleIndex === 0 ? 'Label key' : ''}
                                    isRequired
                                >
                                    <AutoCompleteSelect
                                        selectedOption={labelKey}
                                        onChange={onChangeLabelKey(
                                            scopedResourceSelector,
                                            ruleIndex
                                        )}
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
                                    {rule.values.map(({ value }, valueIndex) => (
                                        <Flex key={value}>
                                            <AutoCompleteSelect
                                                className="pf-u-flex-grow-1 pf-u-w-auto"
                                                selectedOption={value.replace(/.*=/, '')}
                                                onChange={onChangeLabelValue(
                                                    scopedResourceSelector,
                                                    ruleIndex,
                                                    valueIndex
                                                )}
                                            />
                                            <Button
                                                variant="plain"
                                                onClick={() => onDeleteValue(ruleIndex, valueIndex)}
                                            >
                                                <TrashIcon
                                                    style={{ cursor: 'pointer' }}
                                                    color="var(--pf-global--Color--dark-200)"
                                                />
                                            </Button>
                                        </Flex>
                                    ))}
                                </Flex>
                                <Button
                                    className="pf-u-pl-0 pf-u-pt-md"
                                    variant="link"
                                    onClick={() => onAddLabelValue(ruleIndex, labelKey)}
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

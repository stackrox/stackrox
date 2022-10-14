import React from 'react';
import {
    Button,
    Card,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    FormGroup,
    Label,
    Select,
    SelectOption,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';
import cloneDeep from 'lodash/cloneDeep';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Divide } from 'react-feather';
import { SelectorEntityType } from './collections.utils';
import { isByLabelField, isByNameField, ScopedResourceSelector } from './types';

const selectorOptions = ['All', 'ByName', 'ByLabel'] as const;

type RuleSelectorOption = typeof selectorOptions[number];

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return selectorOptions.includes(value as RuleSelectorOption);
}

type AutoCompleteSelectorProps = {
    selectedOption: string;
    onChange: (value: string) => void;
};

/* TODO Implement autocompletion */
function AutoCompleteSelector({ selectedOption, onChange }: AutoCompleteSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        onChange(value);
        closeSelect();
    }

    return (
        <>
            <Select
                variant="typeahead"
                isCreatable
                isOpen={isOpen}
                onToggle={onToggle}
                selections={selectedOption}
                onSelect={onSelect}
            />
        </>
    );
}

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ScopedResourceSelector | null;
    onOptionChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function RuleSelector({ entityType, scopedResourceSelector, onOptionChange }: RuleSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const pluralEntity = pluralize(entityType);

    function onRuleOptionSelect(_, value) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        const selectorMap: Record<RuleSelectorOption, ScopedResourceSelector | null> = {
            All: null,
            ByName: {
                field: entityType,
                rules: [{ operator: 'OR', values: [{ value: '' }] }],
            },
            ByLabel: {
                field: `${entityType} Label`,
                rules: [{ operator: 'OR', values: [{ value: '=' }] }],
            },
        };

        onOptionChange(entityType, selectorMap[value]);
        closeSelect();
    }

    function onChangeNameValue(resourceSelector, ruleIndex, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rules[ruleIndex].values[valueIndex] = { value };
            onOptionChange(entityType, newSelector);
        };
    }

    // TODO Better validation for regex (disallow '=' in user entered values ??)
    function onChangeLabelKey(resourceSelector, ruleIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            const currentValues = newSelector.rules[ruleIndex].values;
            newSelector.rules[ruleIndex].values = currentValues.map((label) => ({
                value: label.value.replace(/.*=/, `${value}=`),
            }));
            onOptionChange(entityType, newSelector);
        };
    }

    function onChangeLabelValue(resourceSelector, ruleIndex, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            const targetValue = newSelector.rules[ruleIndex].values[valueIndex].value;
            newSelector.rules[ruleIndex].values[valueIndex] = {
                value: targetValue.replace(/=.*/, `=${value}`),
            };
            onOptionChange(entityType, newSelector);
        };
    }

    function onAddNameValue() {
        const selector = cloneDeep(scopedResourceSelector);
        const rule = selector?.rules[0];

        // Only add a new form row if there are no blank entries
        if (!rule || !rule.values.every(({ value }) => value)) {
            return;
        }

        selector.rules[0].values.push({ value: '' });
        onOptionChange(entityType, selector);
    }

    function onAddLabelRule() {
        const selector = cloneDeep(scopedResourceSelector);
        const lastRule = selector?.rules[selector?.rules.length - 1];

        // Only add a new form row if there are no blank entries
        if (!lastRule || lastRule.values.every(({ value }) => value === '=')) {
            return;
        }

        selector.rules.push({ operator: 'OR', values: [{ value: '=' }] });
        onOptionChange(entityType, selector);
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
        onOptionChange(entityType, selector);
    }

    function onDeleteValue(ruleIndex: number, valueIndex: number) {
        if (!scopedResourceSelector || !scopedResourceSelector.rules[ruleIndex]) {
            return;
        }

        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rules[ruleIndex].values.length > 1) {
            newSelector.rules[ruleIndex].values.splice(valueIndex, 1);
            onOptionChange(entityType, newSelector);
        } else if (newSelector.rules.length > 1) {
            // This was the last value, so drop the rule
            newSelector.rules.splice(ruleIndex, 1);
            onOptionChange(entityType, newSelector);
        } else {
            // This was the last value in the last rule, so drop the selector
            onOptionChange(entityType, null);
        }
    }

    let selection: RuleSelectorOption = 'All';

    if (!scopedResourceSelector || scopedResourceSelector.rules.length === 0) {
        selection = 'All';
    } else if (isByNameField(scopedResourceSelector.field)) {
        selection = 'ByName';
    } else if (isByLabelField(scopedResourceSelector.field)) {
        selection = 'ByLabel';
    }

    const shouldRenderByNameInputs = scopedResourceSelector && selection === 'ByName';
    const shouldRenderByLabelInputs = scopedResourceSelector && selection === 'ByLabel';

    return (
        <Card>
            <CardBody>
                <Select
                    className={`${selection === 'All' ? '' : 'pf-u-mb-lg'}`}
                    isOpen={isOpen}
                    onToggle={onToggle}
                    selections={selection}
                    onSelect={onRuleOptionSelect}
                >
                    <SelectOption value="All">All {pluralEntity.toLowerCase()}</SelectOption>
                    <SelectOption value="ByName">{pluralEntity} with names matching</SelectOption>
                    <SelectOption value="ByLabel">{pluralEntity} with labels matching</SelectOption>
                </Select>

                {shouldRenderByNameInputs && (
                    <FormGroup label={`${entityType} name`} isRequired>
                        <Flex
                            spaceItems={{ default: 'spaceItemsSm' }}
                            direction={{ default: 'column' }}
                        >
                            {scopedResourceSelector.rules.flatMap((rule) =>
                                rule.values.map(({ value }, index) => (
                                    <Flex key={value}>
                                        <FlexItem grow={{ default: 'grow' }}>
                                            <AutoCompleteSelector
                                                selectedOption={value}
                                                onChange={onChangeNameValue(
                                                    scopedResourceSelector,
                                                    0,
                                                    index
                                                )}
                                            />
                                        </FlexItem>
                                        <TrashIcon
                                            className="pf-u-flex-shrink-1"
                                            style={{ cursor: 'pointer' }}
                                            color="var(--pf-global--Color--dark-200)"
                                            onClick={() => onDeleteValue(0, index)}
                                        />
                                    </Flex>
                                ))
                            )}
                        </Flex>
                        <Button
                            className="pf-u-pl-0 pf-u-pt-md"
                            variant="link"
                            onClick={onAddNameValue}
                        >
                            Add value
                        </Button>
                    </FormGroup>
                )}

                {shouldRenderByLabelInputs && (
                    <>
                        {scopedResourceSelector.rules.map((rule, ruleIndex) => {
                            const labelKey = rule.values[0]?.value?.split('=')[0] ?? '';
                            return (
                                <>
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

                                    <Flex key={labelKey}>
                                        <FlexItem grow={{ default: 'grow' }}>
                                            <Flex>
                                                <FlexItem grow={{ default: 'grow' }}>
                                                    <FormGroup
                                                        label={ruleIndex === 0 ? 'Label key' : ''}
                                                        isRequired
                                                    >
                                                        <AutoCompleteSelector
                                                            selectedOption={labelKey}
                                                            onChange={onChangeLabelKey(
                                                                scopedResourceSelector,
                                                                ruleIndex
                                                            )}
                                                        />
                                                    </FormGroup>
                                                </FlexItem>
                                                <FlexItem
                                                    className="pf-u-pb-xs"
                                                    alignSelf={{ default: 'alignSelfFlexEnd' }}
                                                >
                                                    =
                                                </FlexItem>
                                            </Flex>
                                        </FlexItem>
                                        <FlexItem grow={{ default: 'grow' }}>
                                            <FormGroup
                                                label={ruleIndex === 0 ? 'Label value(s)' : ''}
                                                isRequired
                                            >
                                                <Flex
                                                    spaceItems={{ default: 'spaceItemsSm' }}
                                                    direction={{ default: 'column' }}
                                                >
                                                    {rule.values.map(({ value }, valueIndex) => (
                                                        <Flex key={value}>
                                                            <FlexItem grow={{ default: 'grow' }}>
                                                                <AutoCompleteSelector
                                                                    selectedOption={value.replace(
                                                                        /.*=/,
                                                                        ''
                                                                    )}
                                                                    onChange={onChangeLabelValue(
                                                                        scopedResourceSelector,
                                                                        ruleIndex,
                                                                        valueIndex
                                                                    )}
                                                                />
                                                            </FlexItem>
                                                            <TrashIcon
                                                                style={{ cursor: 'pointer' }}
                                                                color="var(--pf-global--Color--dark-200)"
                                                                onClick={() =>
                                                                    onDeleteValue(
                                                                        ruleIndex,
                                                                        valueIndex
                                                                    )
                                                                }
                                                            />
                                                        </Flex>
                                                    ))}
                                                </Flex>
                                                <Button
                                                    className="pf-u-pl-0 pf-u-pt-md"
                                                    variant="link"
                                                    onClick={() =>
                                                        onAddLabelValue(ruleIndex, labelKey)
                                                    }
                                                >
                                                    Add value
                                                </Button>
                                            </FormGroup>
                                        </FlexItem>
                                    </Flex>
                                </>
                            );
                        })}
                        <Divider component="div" className="pf-u-pt-lg" />
                        <Button
                            className="pf-u-pl-0 pf-u-pt-md"
                            variant="link"
                            onClick={onAddLabelRule}
                        >
                            Add label rule
                        </Button>
                    </>
                )}
            </CardBody>
        </Card>
    );
}

export default RuleSelector;

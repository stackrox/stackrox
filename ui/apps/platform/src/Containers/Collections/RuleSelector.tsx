import React from 'react';
import {
    Button,
    Card,
    CardBody,
    Flex,
    FlexItem,
    FormGroup,
    Select,
    SelectOption,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';
import cloneDeep from 'lodash/cloneDeep';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SelectorEntityType } from './collections.utils';
import {
    isByLabelField,
    isByNameField,
    ScopedResourceSelector,
    ScopedResourceSelectorRule,
} from './types';

const selectorOptions = ['All', 'ByName', 'ByLabel'] as const;

type RuleSelectorOption = typeof selectorOptions[number];

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return selectorOptions.includes(value as RuleSelectorOption);
}

type AutoCompleteSelectorProps = {
    onChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
    entityType: SelectorEntityType;
    resourceSelector: ScopedResourceSelector;
    selectedOption: string;
    index: number;
};

/* TODO Implement autocompletion */
function AutoCompleteSelector({
    onChange,
    entityType,
    resourceSelector,
    selectedOption,
    index,
}: AutoCompleteSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        const newSelector = cloneDeep(resourceSelector);

        newSelector.rules[0].values[index] = { value };

        onChange(entityType, newSelector);
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

    let selection: RuleSelectorOption = 'All';

    if (!scopedResourceSelector) {
        selection = 'All';
    } else if (isByNameField(scopedResourceSelector.field)) {
        selection = 'ByName';
    } else if (isByLabelField(scopedResourceSelector.field)) {
        selection = 'ByLabel';
    }

    function onSelect(_, value) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        const emptyRule: ScopedResourceSelectorRule = {
            operator: 'OR',
            values: [{ value: '' }],
        };

        const selectorMap: Record<RuleSelectorOption, ScopedResourceSelector | null> = {
            All: null,
            ByName: { field: entityType, rules: [emptyRule] },
            ByLabel: { field: `${entityType} Label`, rules: [emptyRule] },
        };

        onOptionChange(entityType, selectorMap[value]);
        closeSelect();
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

    function onAddLabelRule() {
        console.log('add label rule');
    }

    return (
        <Card>
            <CardBody>
                <Select
                    className={`${selection === 'All' ? '' : 'pf-u-mb-lg'}`}
                    isOpen={isOpen}
                    onToggle={onToggle}
                    selections={selection}
                    onSelect={onSelect}
                >
                    <SelectOption value="All">All {pluralEntity.toLowerCase()}</SelectOption>
                    <SelectOption value="ByName">{pluralEntity} with names matching</SelectOption>
                    <SelectOption value="ByLabel">{pluralEntity} with labels matching</SelectOption>
                </Select>

                {scopedResourceSelector &&
                    scopedResourceSelector.rules.length === 1 &&
                    selection === 'ByName' && (
                        <FormGroup label={`${entityType} name`} isRequired>
                            {scopedResourceSelector.rules[0].values.map(({ value }, index) => (
                                <Flex key={value}>
                                    <FlexItem grow={{ default: 'grow' }}>
                                        <AutoCompleteSelector
                                            onChange={onOptionChange}
                                            entityType={entityType}
                                            resourceSelector={scopedResourceSelector}
                                            selectedOption={value}
                                            index={index}
                                        />
                                    </FlexItem>
                                    <TrashIcon
                                        className="pf-u-flex-shrink-1"
                                        style={{ cursor: 'pointer' }}
                                        color="var(--pf-global--Color--dark-200)"
                                        onClick={() => {
                                            const newSelector = cloneDeep(scopedResourceSelector);
                                            newSelector.rules[0]?.values.splice(index, 1);
                                            onOptionChange(entityType, newSelector);
                                        }}
                                    />
                                </Flex>
                            ))}
                            <Button
                                className="pf-u-pl-0 pf-u-pt-md"
                                variant="link"
                                onClick={onAddNameValue}
                            >
                                Add value
                            </Button>
                        </FormGroup>
                    )}

                {scopedResourceSelector && selection === 'ByLabel' && (
                    <>
                        {scopedResourceSelector.rules.map((rule, ruleIndex) => {
                            const labelKey = rule.values[0]?.value?.split('=')[0] ?? '';
                            return (
                                <Flex>
                                    <FormGroup label="Label key" key={labelKey}>
                                        <AutoCompleteSelector
                                            onChange={onOptionChange}
                                            entityType={entityType}
                                            resourceSelector={scopedResourceSelector}
                                            selectedOption={labelKey}
                                            index={ruleIndex}
                                        />
                                    </FormGroup>
                                    <FlexItem>=</FlexItem>
                                    <FormGroup label="Label value(s)">
                                        {rule.values.map(({ value }, valueIndex) => (
                                            <>
                                                <AutoCompleteSelector
                                                    onChange={onOptionChange}
                                                    entityType={entityType}
                                                    resourceSelector={scopedResourceSelector}
                                                    selectedOption={value}
                                                    index={valueIndex}
                                                />
                                                <TrashIcon
                                                    style={{ cursor: 'pointer' }}
                                                    color="var(--pf-global--Color--dark-200)"
                                                    onClick={() => {
                                                        const newSelector =
                                                            cloneDeep(scopedResourceSelector);
                                                        newSelector.rules[ruleIndex]?.values.splice(
                                                            valueIndex,
                                                            1
                                                        );
                                                        onOptionChange(entityType, newSelector);
                                                    }}
                                                />
                                            </>
                                        ))}
                                        <Button
                                            variant="link"
                                            onClick={() => onAddLabelValue(ruleIndex, labelKey)}
                                        >
                                            Add value
                                        </Button>
                                    </FormGroup>
                                </Flex>
                            );
                        })}
                        <Button variant="link" onClick={onAddLabelRule}>
                            Add label rule
                        </Button>
                    </>
                )}
            </CardBody>
        </Card>
    );
}

export default RuleSelector;

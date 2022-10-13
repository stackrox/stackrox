import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';
import cloneDeep from 'lodash/cloneDeep';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { ensureExhaustive } from 'utils/type.utils';
import { TrashIcon } from '@patternfly/react-icons';
import { SelectorEntityType } from './collections.utils';
import { isByLabelSelectorField, isByNameSelectorField, ScopedResourceSelector } from './types';

export const selectorOption = {
    All: 'All',
    ByName: 'ByName',
    ByLabel: 'ByLabel',
} as const;

type RuleSelectorOption = typeof selectorOption[keyof typeof selectorOption];

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return Object.values(selectorOption).includes(value as RuleSelectorOption);
}

type AutoCompleteSelectorProps = {
    onChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
    entityType: SelectorEntityType;
    resourceSelector: ScopedResourceSelector;
    selectedOption: string;
};

function AutoCompleteSelector({
    onChange,
    entityType,
    resourceSelector,
    selectedOption,
}: AutoCompleteSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        const newSelector = cloneDeep(resourceSelector);

        if (newSelector.rules.length > 0) {
            newSelector.rules[0].values.push({ value });
        } else {
            newSelector.rules = [{ operator: 'OR', values: [{ value }] }];
        }

        onChange(entityType, newSelector);
        closeSelect();
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
            >
                <SelectOption value="test">test</SelectOption>
                <SelectOption value="test2">test2</SelectOption>
                <SelectOption value="test3">test3</SelectOption>
            </Select>
        </>
    );
}

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    selectedOption: ScopedResourceSelector | null;
    onOptionChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function RuleSelector({ entityType, selectedOption, onOptionChange }: RuleSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const pluralEntity = pluralize(entityType);

    function onSelect(_, value) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        let selector: ScopedResourceSelector | null = null;

        switch (value) {
            case 'All':
                selector = null;
                break;
            case 'ByName':
                selector = { field: entityType, rules: [] };
                break;
            case 'ByLabel':
                selector = { field: `${entityType} Label`, rules: [] };
                break;
            default:
                ensureExhaustive(value);
        }

        onOptionChange(entityType, selector);
        closeSelect();
    }

    let selections: RuleSelectorOption = selectorOption.All;

    if (!selectedOption) {
        selections = selectorOption.All;
    } else if (isByNameSelectorField(selectedOption.field)) {
        selections = selectorOption.ByName;
    } else if (isByLabelSelectorField(selectedOption.field)) {
        selections = selectorOption.ByLabel;
    }

    return (
        <>
            <Select isOpen={isOpen} onToggle={onToggle} selections={selections} onSelect={onSelect}>
                <SelectOption value={selectorOption.All}>
                    All {pluralEntity.toLowerCase()}
                </SelectOption>
                <SelectOption value={selectorOption.ByName}>
                    {pluralEntity} with names matching
                </SelectOption>
                <SelectOption value={selectorOption.ByLabel}>
                    {pluralEntity} with labels matching
                </SelectOption>
            </Select>

            <>
                {selectedOption && selections === selectorOption.ByName && (
                    <>
                        {selectedOption.rules[0]?.values?.map(({ value }, index) => (
                            <>
                                <AutoCompleteSelector
                                    onChange={onOptionChange}
                                    entityType={entityType}
                                    resourceSelector={selectedOption}
                                    selectedOption={value}
                                />
                                <TrashIcon
                                    onClick={() => {
                                        const newSelector = cloneDeep(selectedOption);
                                        newSelector.rules[0]?.values.splice(index, 1);
                                        onOptionChange(entityType, newSelector);
                                    }}
                                />
                            </>
                        ))}
                        <AutoCompleteSelector
                            onChange={onOptionChange}
                            entityType={entityType}
                            resourceSelector={selectedOption}
                            selectedOption=""
                        />
                    </>
                )}
            </>
            {selections === selectorOption.ByLabel && <></>}
        </>
    );
}

export default RuleSelector;

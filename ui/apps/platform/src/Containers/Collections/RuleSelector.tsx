import React from 'react';
import { Form, FormGroup, Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { ensureExhaustive } from 'utils/type.utils';
import { SelectorEntityType } from './collections.utils';
import { isByLabelSelectorField, isByNameSelectorField, ScopedResourceSelector } from './types';

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    selectedOption: ScopedResourceSelector | null;
    onOptionChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

export const selectorOption = {
    All: 'All',
    ByName: 'ByName',
    ByLabel: 'ByLabel',
} as const;

type RuleSelectorOption = typeof selectorOption[keyof typeof selectorOption];

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return Object.values(selectorOption).includes(value as RuleSelectorOption);
}

function AutoCompleteSelector({ onChange }) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        closeSelect();
    }

    return (
        <>
            <Select isOpen={isOpen} onToggle={onToggle} selections={[]} onSelect={onSelect}>
                <SelectOption value="test">test</SelectOption>
                <SelectOption value="test2">test2</SelectOption>
            </Select>
        </>
    );
}

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
                break;
            case 'ByName':
                selector = {
                    field: entityType,
                    rules: [{ operator: 'OR', values: [{ value: 'test1' }] }],
                };
                break;
            case 'ByLabel':
                selector = {
                    field: `${entityType} Label`,
                    rules: [{ operator: 'OR', values: [{ value: 'test2' }] }],
                };
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

            {selections === selectorOption.ByName && (
                <AutoCompleteSelector onChange={onOptionChange} />
            )}
            {selections === selectorOption.ByLabel && <></>}
        </>
    );
}

export default RuleSelector;

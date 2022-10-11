import React from 'react';
import { Form, FormGroup, Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SelectorEntityType } from './collections.utils';
import { isByLabelSelectorField, isByNameSelectorField, ScopedResourceSelector } from './types';

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    selectedOption: ScopedResourceSelector;
    onOptionChange: (option: RuleSelectorOption) => void;
};

export const SelectorOption = {
    All: 'All',
    ByName: 'ByName',
    ByLabel: 'ByLabel',
} as const;

type RuleSelectorOption = typeof SelectorOption[keyof typeof SelectorOption];

function AutoCompleteSelector() {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        closeSelect();
    }

    return (
        <>
            <Select isOpen={isOpen} onToggle={onToggle} selections={[]} onSelect={onSelect} />
        </>
    );
}

// TODO - Evaluate whether or not this makes sense to move into Component/PatternFly for general use
function RuleSelector({ entityType, selectedOption, onOptionChange }: RuleSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const pluralEntity = pluralize(entityType);

    function onSelect(_, value) {
        onOptionChange(value);
        closeSelect();
    }

    let selections: RuleSelectorOption = SelectorOption.All;

    if (!selectedOption) {
        selections = SelectorOption.All;
    } else if (isByNameSelectorField(selectedOption.field)) {
        selections = SelectorOption.ByName;
    } else if (isByLabelSelectorField(selectedOption.field)) {
        selections = SelectorOption.ByLabel;
    }

    return (
        <>
            <Select isOpen={isOpen} onToggle={onToggle} selections={selections} onSelect={onSelect}>
                <SelectOption value={SelectorOption.All}>
                    All {pluralEntity.toLowerCase()}
                </SelectOption>
                <SelectOption value={SelectorOption.ByName}>
                    {pluralEntity} with names matching
                </SelectOption>
                <SelectOption value={SelectorOption.ByLabel}>
                    {pluralEntity} with labels matching
                </SelectOption>
            </Select>

            {selections === SelectorOption.ByName && <></>}
            {selections === SelectorOption.ByLabel && <></>}
        </>
    );
}

export default RuleSelector;

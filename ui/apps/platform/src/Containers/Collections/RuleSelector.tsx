import React from 'react';
import { Form, FormGroup, Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SelectorRule } from 'services/CollectionsService';
import { SelectorEntityType } from './collections.utils';

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    selectedOption: RuleSelectorOption;
    onOptionChange: (option: RuleSelectorOption) => void;
    onRulesChange: (rules: SelectorRule[]) => void;
};

export const SelectorOption = {
    All: 'All',
    ByName: 'ByName',
    ByLabel: 'ByLabel',
} as const;

export type RuleSelectorOption = SelectOption[keyof SelectOption];

function AutoCompleteSelector() {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();

    function onSelect(_, value) {
        onOptionChange(value);
        closeSelect();
    }

    return (
        <>
            <Select
                isOpen={isOpen}
                onToggle={onToggle}
                selections={selectedOption}
                onSelect={onSelect}
            >
                <SelectOption>test</SelectOption>
            </Select>
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

    return (
        <>
            <Form>
                <Select
                    isOpen={isOpen}
                    onToggle={onToggle}
                    selections={selectedOption}
                    onSelect={onSelect}
                >
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
                {selectedOption === SelectorOption.ByName && (
                    <FormGroup label={`${entityType} name`}>
                        <AutoCompleteSelector />
                    </FormGroup>
                )}
                {selectedOption === SelectorOption.ByLabel && <></>}
            </Form>
        </>
    );
}

export default RuleSelector;

import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SelectorEntityType } from '../collections.utils';
import { isByLabelField, isByNameField, ScopedResourceSelector } from '../types';
import ByNameSelector from './ByNameSelector';
import ByLabelSelector from './ByLabelSelector';

const selectorOptions = ['All', 'ByName', 'ByLabel'] as const;

type RuleSelectorOption = typeof selectorOptions[number];

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return selectorOptions.includes(value as RuleSelectorOption);
}

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ScopedResourceSelector | null;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function RuleSelector({ entityType, scopedResourceSelector, handleChange }: RuleSelectorProps) {
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

        handleChange(entityType, selectorMap[value]);
        closeSelect();
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
        <div
            className="pf-u-p-lg"
            style={{ border: '1px solid var(--pf-global--BorderColor--100' }}
        >
            <Select
                toggleAriaLabel={`Select ${pluralEntity.toLowerCase()} by name or label`}
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
                <ByNameSelector
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                />
            )}

            {shouldRenderByLabelInputs && (
                <ByLabelSelector
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                />
            )}
        </div>
    );
}

export default RuleSelector;

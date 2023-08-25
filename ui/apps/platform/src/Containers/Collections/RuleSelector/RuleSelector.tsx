import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';
import { FormikErrors } from 'formik';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import {
    RuleSelectorOption,
    ScopedResourceSelector,
    SelectorEntityType,
    selectorOptions,
} from '../types';
import ByNameSelector from './ByNameSelector';
import ByLabelSelector from './ByLabelSelector';

import './RuleSelector.css';

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return selectorOptions.includes(value as RuleSelectorOption);
}

const placeholders = {
    ByName: {
        Deployment: 'nginx-deployment',
        Namespace: 'payments',
        Cluster: 'production',
    },
    ByLabel: {
        Deployment: 'app=nginx',
        Namespace: 'team=payments',
        Cluster: 'environment=production',
    },
};

export type RuleSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ScopedResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    validationErrors: FormikErrors<ScopedResourceSelector> | undefined;
    isDisabled?: boolean;
};

function RuleSelector({
    entityType,
    scopedResourceSelector,
    handleChange,
    validationErrors,
    isDisabled = false,
}: RuleSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const pluralEntity = pluralize(entityType);

    function onRuleOptionSelect(_, value) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        const selectorMap: Record<RuleSelectorOption, ScopedResourceSelector> = {
            All: { type: 'All' },
            ByName: {
                type: 'ByName',
                field: entityType,
                rule: { operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] },
            },
            ByLabel: {
                type: 'ByLabel',
                field: `${entityType} Label`,
                rules: [{ operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] }],
            },
        };

        handleChange(entityType, selectorMap[value]);
        closeSelect();
    }

    const selection = scopedResourceSelector.type;

    return (
        <div className="rule-selector">
            <Select
                toggleAriaLabel={`Select ${pluralEntity.toLowerCase()} by name or label`}
                isOpen={isOpen}
                onToggle={onToggle}
                selections={selection}
                onSelect={onRuleOptionSelect}
                isDisabled={isDisabled}
            >
                <SelectOption value="All">All {pluralEntity.toLowerCase()}</SelectOption>
                <SelectOption value="ByName">{pluralEntity} with names matching</SelectOption>
                <SelectOption value="ByLabel">
                    {pluralEntity} with labels matching exactly
                </SelectOption>
            </Select>

            {scopedResourceSelector.type === 'ByName' && (
                <ByNameSelector
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                    placeholder={placeholders.ByName[entityType]}
                    validationErrors={validationErrors}
                    isDisabled={isDisabled}
                />
            )}

            {scopedResourceSelector.type === 'ByLabel' && (
                <ByLabelSelector
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                    placeholder={placeholders.ByLabel[entityType]}
                    validationErrors={validationErrors}
                    isDisabled={isDisabled}
                />
            )}
        </div>
    );
}

export default RuleSelector;

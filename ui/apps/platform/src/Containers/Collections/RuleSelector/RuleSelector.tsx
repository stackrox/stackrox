import { SelectOption } from '@patternfly/react-core';
import pluralize from 'pluralize';
import type { FormikErrors } from 'formik';

import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { selectorOptions } from '../types';
import type { RuleSelectorOption, ScopedResourceSelector, SelectorEntityType } from '../types';
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
    const pluralEntity = pluralize(entityType);

    function onRuleOptionSelect(_id: string, value: string) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        const selectorMap: Record<RuleSelectorOption, ScopedResourceSelector> = {
            NoneSpecified: { type: 'NoneSpecified' },
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
    }

    const selection = scopedResourceSelector.type;
    const selectorId = `rule-selector-${entityType.toLowerCase()}`;

    return (
        <div className="rule-selector">
            <SelectSingle
                id={selectorId}
                toggleAriaLabel={`Select ${pluralEntity.toLowerCase()} by name or label`}
                value={selection}
                handleSelect={onRuleOptionSelect}
                isDisabled={isDisabled}
            >
                <SelectOption value="NoneSpecified">
                    No {pluralEntity.toLowerCase()} specified
                </SelectOption>
                <SelectOption value="ByName">{pluralEntity} with names matching</SelectOption>
                <SelectOption value="ByLabel">
                    {pluralEntity} with labels matching exactly
                </SelectOption>
            </SelectSingle>

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

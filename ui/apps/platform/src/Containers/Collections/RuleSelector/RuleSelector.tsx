import React from 'react';
import { Select, SelectOption, SelectOptionProps } from '@patternfly/react-core';
import pluralize from 'pluralize';
import { FormikErrors } from 'formik';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import ResourceIcon from 'Components/PatternFly/ResourceIcon';
import {
    Collection,
    RuleSelectorOption,
    ScopedResourceSelector,
    SelectorEntityType,
    selectorOptions,
} from '../types';
import ByNameSelector from './ByNameSelector';
import ByLabelSelector from './ByLabelSelector';

function isRuleSelectorOption(value: string): value is RuleSelectorOption {
    return selectorOptions.includes(value as RuleSelectorOption);
}

export type RuleSelectorProps = {
    collection: Collection;
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
    collection,
    entityType,
    scopedResourceSelector,
    handleChange,
    validationErrors,
    isDisabled = false,
}: RuleSelectorProps) {
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const pluralEntity = pluralize(entityType);

    const OptionComponent = (props: SelectOptionProps) => (
        <div {...props}>
            <ResourceIcon kind={entityType} />
            {props.value}
        </div>
    );

    function onRuleOptionSelect(_, value) {
        if (!isRuleSelectorOption(value)) {
            return;
        }

        const selectorMap: Record<RuleSelectorOption, ScopedResourceSelector> = {
            All: { type: 'All' },
            ByName: {
                type: 'ByName',
                field: entityType,
                rule: { operator: 'OR', values: [''] },
            },
            ByLabel: {
                type: 'ByLabel',
                field: `${entityType} Label`,
                rules: [{ operator: 'OR', key: '', values: [''] }],
            },
        };

        handleChange(entityType, selectorMap[value]);
        closeSelect();
    }

    const selection = scopedResourceSelector.type;

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
                isDisabled={isDisabled}
            >
                <SelectOption value="All">All {pluralEntity.toLowerCase()}</SelectOption>
                <SelectOption value="ByName">{pluralEntity} with names matching</SelectOption>
                <SelectOption value="ByLabel">{pluralEntity} with labels matching</SelectOption>
            </Select>

            {scopedResourceSelector.type === 'ByName' && (
                <ByNameSelector
                    collection={collection}
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                    validationErrors={validationErrors}
                    isDisabled={isDisabled}
                    OptionComponent={OptionComponent}
                />
            )}

            {scopedResourceSelector.type === 'ByLabel' && (
                <ByLabelSelector
                    entityType={entityType}
                    scopedResourceSelector={scopedResourceSelector}
                    handleChange={handleChange}
                    validationErrors={validationErrors}
                    isDisabled={isDisabled}
                />
            )}
        </div>
    );
}

export default RuleSelector;

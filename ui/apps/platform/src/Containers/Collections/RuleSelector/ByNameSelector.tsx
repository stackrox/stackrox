import React from 'react';
import { Button, SelectOption, TextInput, ValidatedOptions } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import cloneDeep from 'lodash/cloneDeep';

import { FormikErrors } from 'formik';
import useIndexKey from 'hooks/useIndexKey';
import { AutoCompleteSelect } from './AutoCompleteSelect';
import {
    ByNameMatchType,
    ByNameResourceSelector,
    ClientCollection,
    ScopedResourceSelector,
    SelectorEntityType,
} from '../types';
import { NameMatchTypeSelect } from './MatchTypeSelect';

export type ByNameSelectorProps = {
    collection: ClientCollection;
    entityType: SelectorEntityType;
    scopedResourceSelector: ByNameResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    placeholder: string;
    validationErrors: FormikErrors<ByNameResourceSelector> | undefined;
    isDisabled: boolean;
};

function ByNameSelector({
    collection,
    entityType,
    scopedResourceSelector,
    handleChange,
    placeholder,
    validationErrors,
    isDisabled,
}: ByNameSelectorProps) {
    const { keyFor, invalidateIndexKeys } = useIndexKey();
    const lowerCaseEntity = entityType.toLowerCase();

    function onAddValue() {
        const selector = cloneDeep(scopedResourceSelector);
        // Only add a new form row if there are no blank entries
        if (selector.rule.values.every(({ value }) => value)) {
            selector.rule.values.push({ value: '', matchType: 'EXACT' });
            handleChange(entityType, selector);
        }
    }

    function onChangeMatchType(resourceSelector: ByNameResourceSelector, valueIndex: number) {
        return (value: ByNameMatchType) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex].matchType = value;
            handleChange(entityType, newSelector);
        };
    }

    function onChangeValue(resourceSelector, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex].value = value;
            handleChange(entityType, newSelector);
        };
    }

    function onDeleteValue(valueIndex: number) {
        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rule.values.length > 1) {
            newSelector.rule.values.splice(valueIndex, 1);
            invalidateIndexKeys();
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the rule, so drop the selector
            handleChange(entityType, { type: 'All' });
        }
    }

    return (
        <>
            <div className="rule-selector-list">
                {scopedResourceSelector.rule.values.map(({ matchType, value }, index) => {
                    const inputId = `${entityType}-name-value-${index}`;
                    const inputAriaLabel = `Select value ${index + 1} of ${
                        scopedResourceSelector.rule.values.length
                    } for the ${lowerCaseEntity} name`;
                    const inputClassName = 'pf-u-flex-grow-1 pf-u-w-auto';
                    const inputOnChange = onChangeValue(scopedResourceSelector, index);
                    const inputValidated = validationErrors?.rule?.values?.[index]
                        ? ValidatedOptions.error
                        : ValidatedOptions.default;

                    return (
                        <div className="rule-selector-list-item" key={keyFor(index)}>
                            <div className="rule-selector-match-type-select">
                                <NameMatchTypeSelect
                                    selected={matchType}
                                    isDisabled={isDisabled}
                                    onChange={onChangeMatchType(scopedResourceSelector, index)}
                                >
                                    <SelectOption value="EXACT">An exact value of</SelectOption>
                                    <SelectOption value="REGEX">A regex value of</SelectOption>
                                </NameMatchTypeSelect>
                            </div>
                            <div className="rule-selector-name-value-input">
                                {matchType === 'REGEX' ? (
                                    <TextInput
                                        id={inputId}
                                        aria-label={inputAriaLabel}
                                        placeholder={`^${placeholder}$`}
                                        className={inputClassName}
                                        onChange={inputOnChange}
                                        validated={inputValidated}
                                        value={value}
                                        isDisabled={isDisabled}
                                    />
                                ) : (
                                    <AutoCompleteSelect
                                        id={inputId}
                                        entityType={entityType}
                                        typeAheadAriaLabel={inputAriaLabel}
                                        className={inputClassName}
                                        onChange={inputOnChange}
                                        placeholder={placeholder}
                                        validated={inputValidated}
                                        selectedOption={value}
                                        isDisabled={isDisabled}
                                        collection={collection}
                                        autocompleteField={entityType}
                                    />
                                )}
                            </div>
                            {!isDisabled && (
                                <Button
                                    aria-label={`Delete ${value}`}
                                    variant="plain"
                                    onClick={() => onDeleteValue(index)}
                                >
                                    <TrashIcon
                                        className="pf-u-flex-shrink-1"
                                        style={{ cursor: 'pointer' }}
                                        color="var(--pf-global--Color--dark-200)"
                                    />
                                </Button>
                            )}
                        </div>
                    );
                })}
            </div>
            {!isDisabled && (
                <Button
                    aria-label={`Add ${lowerCaseEntity} name value`}
                    className="rule-selector-add-value-button"
                    variant="link"
                    onClick={onAddValue}
                >
                    OR...
                </Button>
            )}
        </>
    );
}

export default ByNameSelector;

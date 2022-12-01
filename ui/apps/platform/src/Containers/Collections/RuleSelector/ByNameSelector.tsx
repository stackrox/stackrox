import React, { ReactNode, useCallback } from 'react';
import { Button, Flex, FormGroup, ValidatedOptions } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import cloneDeep from 'lodash/cloneDeep';

import { FormikErrors } from 'formik';
import useIndexKey from 'hooks/useIndexKey';
import { getCollectionAutoComplete } from 'services/CollectionsService';
import { AutoCompleteSelect } from './AutoCompleteSelect';
import {
    ByNameResourceSelector,
    Collection,
    ScopedResourceSelector,
    SelectorEntityType,
} from '../types';
import { generateRequest } from '../converter';

export type ByNameSelectorProps = {
    collection: Collection;
    entityType: SelectorEntityType;
    scopedResourceSelector: ByNameResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector
    ) => void;
    validationErrors: FormikErrors<ByNameResourceSelector> | undefined;
    isDisabled: boolean;
    OptionComponent: ReactNode;
};

function ByNameSelector({
    collection,
    entityType,
    scopedResourceSelector,
    handleChange,
    validationErrors,
    isDisabled,
    OptionComponent,
}: ByNameSelectorProps) {
    const { keyFor, invalidateIndexKeys } = useIndexKey();
    const lowerCaseEntity = entityType.toLowerCase();
    const autocompleteProvider = useCallback(
        (search: string) => {
            const req = generateRequest(collection);
            return getCollectionAutoComplete(req.resourceSelectors, entityType, search);
        },
        [collection, entityType]
    );

    function onAddValue() {
        const selector = cloneDeep(scopedResourceSelector);
        // Only add a new form row if there are no blank entries
        if (selector.rule.values.every((value) => value)) {
            selector.rule.values.push('');
            handleChange(entityType, selector);
        }
    }

    function onChangeValue(resourceSelector, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex] = value;
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
        <FormGroup fieldId={`${entityType}-name-value`} label={`${entityType} name`} isRequired>
            <Flex spaceItems={{ default: 'spaceItemsSm' }} direction={{ default: 'column' }}>
                {scopedResourceSelector.rule.values.map((value, index) => (
                    <Flex key={keyFor(index)}>
                        <AutoCompleteSelect
                            id={`${entityType}-name-value-${index}`}
                            typeAheadAriaLabel={`Select value ${index + 1} of ${
                                scopedResourceSelector.rule.values.length
                            } for the ${lowerCaseEntity} name`}
                            className="pf-u-flex-grow-1 pf-u-w-auto"
                            selectedOption={value}
                            onChange={onChangeValue(scopedResourceSelector, index)}
                            validated={
                                validationErrors?.rule?.values?.[index]
                                    ? ValidatedOptions.error
                                    : ValidatedOptions.default
                            }
                            isDisabled={isDisabled}
                            autocompleteProvider={autocompleteProvider}
                            OptionComponent={OptionComponent}
                        />
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
                    </Flex>
                ))}
            </Flex>
            {!isDisabled && (
                <Button
                    aria-label={`Add ${lowerCaseEntity} name value`}
                    className="pf-u-pl-0 pf-u-pt-md"
                    variant="link"
                    onClick={onAddValue}
                >
                    Add value
                </Button>
            )}
        </FormGroup>
    );
}

export default ByNameSelector;

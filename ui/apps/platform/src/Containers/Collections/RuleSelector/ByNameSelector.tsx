import React from 'react';
import { Button, Flex, FormGroup } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import cloneDeep from 'lodash/cloneDeep';

import { AutoCompleteSelect } from './AutoCompleteSelect';
import { ByNameResourceSelector, ScopedResourceSelector, SelectorEntityType } from '../types';

export type ByNameSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ByNameResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function ByNameSelector({ entityType, scopedResourceSelector, handleChange }: ByNameSelectorProps) {
    function onAddValue() {
        const selector = cloneDeep(scopedResourceSelector);
        // Only add a new form row if there are no blank entries
        if (!selector.rule.values.every((value) => value)) {
            return;
        }

        selector.rule.values.push('');
        handleChange(entityType, selector);
    }

    function onChangeValue(resourceSelector, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rule.values[valueIndex] = value;
            handleChange(entityType, newSelector);
        };
    }

    function onDeleteValue(valueIndex: number) {
        if (!scopedResourceSelector || !scopedResourceSelector.rule) {
            return;
        }

        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rule.values.length > 1) {
            newSelector.rule.values.splice(valueIndex, 1);
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the rule, so drop the selector
            handleChange(entityType, null);
        }
    }

    return (
        <FormGroup label={`${entityType} name`} isRequired>
            <Flex spaceItems={{ default: 'spaceItemsSm' }} direction={{ default: 'column' }}>
                {scopedResourceSelector.rule.values.map((value, index) => (
                    <Flex key={value}>
                        <AutoCompleteSelect
                            typeAheadAriaLabel={`Select a value for the ${entityType.toLowerCase()} name`}
                            className="pf-u-flex-grow-1 pf-u-w-auto"
                            selectedOption={value}
                            onChange={onChangeValue(scopedResourceSelector, index)}
                        />
                        <Button variant="plain" onClick={() => onDeleteValue(index)}>
                            <TrashIcon
                                aria-label={`Delete ${value}`}
                                className="pf-u-flex-shrink-1"
                                style={{ cursor: 'pointer' }}
                                color="var(--pf-global--Color--dark-200)"
                            />
                        </Button>
                    </Flex>
                ))}
            </Flex>
            <Button className="pf-u-pl-0 pf-u-pt-md" variant="link" onClick={onAddValue}>
                Add value
            </Button>
        </FormGroup>
    );
}

export default ByNameSelector;

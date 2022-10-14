import React from 'react';
import { Button, Flex, FormGroup } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { cloneDeep } from 'lodash';

import { AutoCompleteSelect } from './AutoCompleteSelect';
import { ScopedResourceSelector, SelectorEntityType } from '../types';

export type ByNameSelectorProps = {
    entityType: SelectorEntityType;
    scopedResourceSelector: ScopedResourceSelector;
    handleChange: (
        entityType: SelectorEntityType,
        scopedResourceSelector: ScopedResourceSelector | null
    ) => void;
};

function ByNameSelector({ entityType, scopedResourceSelector, handleChange }: ByNameSelectorProps) {
    function onAddValue() {
        const selector = cloneDeep(scopedResourceSelector);
        const rule = selector?.rules[0];

        // Only add a new form row if there are no blank entries
        if (!rule || !rule.values.every(({ value }) => value)) {
            return;
        }

        selector.rules[0].values.push({ value: '' });
        handleChange(entityType, selector);
    }

    function onChangeValue(resourceSelector, ruleIndex, valueIndex) {
        return (value: string) => {
            const newSelector = cloneDeep(resourceSelector);
            newSelector.rules[ruleIndex].values[valueIndex] = { value };
            handleChange(entityType, newSelector);
        };
    }

    function onDeleteValue(ruleIndex: number, valueIndex: number) {
        if (!scopedResourceSelector || !scopedResourceSelector.rules[ruleIndex]) {
            return;
        }

        const newSelector = cloneDeep(scopedResourceSelector);

        if (newSelector.rules[ruleIndex].values.length > 1) {
            newSelector.rules[ruleIndex].values.splice(valueIndex, 1);
            handleChange(entityType, newSelector);
        } else if (newSelector.rules.length > 1) {
            // This was the last value, so drop the rule
            newSelector.rules.splice(ruleIndex, 1);
            handleChange(entityType, newSelector);
        } else {
            // This was the last value in the last rule, so drop the selector
            handleChange(entityType, null);
        }
    }

    return (
        <FormGroup label={`${entityType} name`} isRequired>
            <Flex spaceItems={{ default: 'spaceItemsSm' }} direction={{ default: 'column' }}>
                {scopedResourceSelector.rules[0]?.values.map(({ value }, index) => (
                    <Flex key={value}>
                        <AutoCompleteSelect
                            className="pf-u-flex-grow-1 pf-u-w-auto"
                            selectedOption={value}
                            onChange={onChangeValue(scopedResourceSelector, 0, index)}
                        />
                        <TrashIcon
                            className="pf-u-flex-shrink-1"
                            style={{ cursor: 'pointer' }}
                            color="var(--pf-global--Color--dark-200)"
                            onClick={() => onDeleteValue(0, index)}
                        />
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

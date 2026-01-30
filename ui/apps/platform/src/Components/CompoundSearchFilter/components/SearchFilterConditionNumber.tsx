import { useState } from 'react';
import type { FormEvent } from 'react';
import { Button, NumberInput, SelectOption, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import clamp from 'lodash/clamp';

import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';
import { conditionMap, conditions } from '../utils/utils';

import SimpleSelect from './SimpleSelect';

export type ConditionNumber = { condition: string; number: number };

// Move to attribute if we ever need to support number other than CVSS.
const minValue = 0;
const maxValue = 10;
const incrementValue = 0.1;

function roundToOneDecimalPlace(value: number) {
    return parseFloat(value.toFixed(1));
}

export type SearchFilterConditionNumberProps = {
    attribute: GenericSearchFilterAttribute;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function SearchFilterConditionNumber({ attribute, onSearch }: SearchFilterConditionNumberProps) {
    const { searchTerm: category } = attribute;

    const [conditionExternal, setConditionExternal] = useState(conditions[0]);
    const [value, setValue] = useState(minValue);

    const updateNumber = (operation: 'minus' | 'plus') => {
        let incrementedNumber = value;
        if (operation === 'minus') {
            incrementedNumber -= incrementValue;
        } else if (operation === 'plus') {
            incrementedNumber += incrementValue;
        }
        const roundedNumber = roundToOneDecimalPlace(incrementedNumber);
        const normalizedNumber = clamp(roundedNumber, minValue, maxValue);
        setValue(normalizedNumber);
    };

    const onMinus = () => updateNumber('minus');

    const onPlus = () => updateNumber('plus');

    return (
        <>
            <ToolbarItem>
                <SimpleSelect
                    value={conditionExternal}
                    onChange={(conditionSelected) =>
                        setConditionExternal(conditionSelected as (typeof conditions)[number])
                    }
                    ariaLabelMenu="Condition selector menu"
                    ariaLabelToggle="Condition selector toggle"
                >
                    {conditions.map((condition) => {
                        return (
                            <SelectOption key={condition} value={condition}>
                                {condition}
                            </SelectOption>
                        );
                    })}
                </SimpleSelect>
            </ToolbarItem>
            <ToolbarItem>
                <NumberInput
                    inputAriaLabel="Condition value input"
                    value={value}
                    min={minValue}
                    max={maxValue}
                    onChange={(event: FormEvent<HTMLInputElement>) => {
                        const { value: valueChanged } = event.target as HTMLInputElement;
                        setValue(Number(valueChanged));
                    }}
                    onBlur={(event: FormEvent<HTMLInputElement>) => {
                        const target = event.target as HTMLInputElement;
                        const normalizedNumber = clamp(
                            Number.isNaN(+target.value)
                                ? 0
                                : roundToOneDecimalPlace(Number(target.value)),
                            minValue,
                            maxValue
                        );
                        setValue(normalizedNumber);
                    }}
                    onMinus={onMinus}
                    onPlus={onPlus}
                    minusBtnAriaLabel="Condition value minus button"
                    plusBtnAriaLabel="Condition value plus button"
                />
            </ToolbarItem>
            <ToolbarItem>
                <Button
                    icon={<ArrowRightIcon />}
                    variant="control"
                    aria-label="Apply condition and number input to search"
                    onClick={() => {
                        const conditionInternal = conditionMap[conditionExternal];
                        if (conditionInternal) {
                            onSearch([
                                {
                                    action: 'APPEND',
                                    category,
                                    value: `${conditionInternal}${value}`,
                                },
                            ]);
                        }
                    }}
                ></Button>
            </ToolbarItem>
        </>
    );
}

export default SearchFilterConditionNumber;

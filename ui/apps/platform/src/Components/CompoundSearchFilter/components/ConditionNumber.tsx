import React from 'react';
import { Button, NumberInput, SelectOption } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import clamp from 'lodash/clamp';

import { ensureString } from 'utils/ensure';
import { conditions } from '../utils/utils';

import SimpleSelect from './SimpleSelect';

export type ConditionNumber = { condition: string; number: number };

export type ConditionNumberProps = {
    value: ConditionNumber;
    onChange: (value: { condition: string; number: number }) => void;
    onSearch: (value: { condition: string; number: number }) => void;
};

const minValue = 0;
const maxValue = 10;
const incrementValue = 0.1;

function roundToOneDecimalPlace(value: number) {
    return parseFloat(value.toFixed(1));
}

function ConditionNumber({ value, onChange, onSearch }: ConditionNumberProps) {
    const updateNumber = (operation: 'minus' | 'plus') => {
        let incrementedNumber = value.number || 0;
        if (operation === 'minus') {
            incrementedNumber -= incrementValue;
        } else if (operation === 'plus') {
            incrementedNumber += incrementValue;
        }
        const roundedNumber = roundToOneDecimalPlace(incrementedNumber);
        const normalizedNumber = clamp(roundedNumber, minValue, maxValue);
        onChange({
            ...value,
            number: normalizedNumber,
        });
    };

    const onMinus = () => updateNumber('minus');

    const onPlus = () => updateNumber('plus');

    return (
        <>
            <SimpleSelect
                value={value.condition || conditions[0]}
                onChange={(val) =>
                    onChange({
                        ...value,
                        condition: ensureString(val),
                    })
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
            <NumberInput
                inputAriaLabel="Condition value input"
                value={value.number || minValue}
                min={0}
                max={10}
                onChange={(event: React.FormEvent<HTMLInputElement>) => {
                    const { value: newNumber } = event.target as HTMLInputElement;
                    onChange({
                        ...value,
                        number: Number(newNumber),
                    });
                }}
                onBlur={(event: React.FormEvent<HTMLInputElement>) => {
                    const target = event.target as HTMLInputElement;
                    const normalizedNumber = clamp(
                        Number.isNaN(+target.value)
                            ? 0
                            : roundToOneDecimalPlace(Number(target.value)),
                        minValue,
                        maxValue
                    );
                    onChange({
                        ...value,
                        number: normalizedNumber,
                    });
                }}
                onMinus={onMinus}
                onPlus={onPlus}
                minusBtnAriaLabel="Condition value minus button"
                plusBtnAriaLabel="Condition value plus button"
            />
            <Button
                variant="control"
                aria-label="Apply condition and number input to search"
                onClick={() => {
                    onSearch(value);
                }}
            >
                <ArrowRightIcon />
            </Button>
        </>
    );
}

export default ConditionNumber;

import React, { useEffect, useState } from 'react';
import { Button, NumberInput, SelectOption } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { ensureString } from '../utils/utils';

import SimpleSelect from './SimpleSelect';

export type ConditionNumberProps = {
    value: string;
    onSearch: (value: string) => void;
};

const conditionMap = {
    'Is greater than': '>',
    'Is greater than or equal to': '>=',
    'Is equal to': '=',
    'Is less than or equal to': '<=',
    'Is less than': '<',
};

const conditions = Object.keys(conditionMap);

const minValue = 0;
const maxValue = 10;
const incrementValue = 0.1;

function roundToOneDecimalPlace(value: number) {
    return parseFloat(value.toFixed(1));
}

const normalizeBetween = (value: number, min: number, max: number) => {
    if (min !== undefined && max !== undefined) {
        return Math.max(Math.min(value, max), min);
    }
    if (value <= min) {
        return min;
    }
    if (value >= max) {
        return max;
    }
    return value;
};

function getDefaultCondition() {
    return conditions[0];
}

function getDefaultNumber() {
    return minValue;
}

function valueToConditionNumber(value: string) {
    const [newConditionValue, newNumber] = value.split(' ');
    const newCondition =
        conditions.find((condition) => {
            return conditionMap[condition] === newConditionValue;
        }) || conditions[0];
    return {
        newCondition,
        newNumber: Number(newNumber) || minValue,
    };
}

function conditionNumberToValue(conditionLabel: string, number: number): string {
    const conditionValue = conditionMap[conditionLabel];
    const newValue = `${conditionValue} ${number}`;
    return newValue;
}

function ConditionNumber({ value, onSearch }: ConditionNumberProps) {
    const [selectedCondition, setSelectedCondition] = useState(() => getDefaultCondition());
    const [number, setNumber] = useState(() => getDefaultNumber());

    useEffect(() => {
        const { newCondition, newNumber } = valueToConditionNumber(value);
        setSelectedCondition(newCondition);
        setNumber(newNumber);
    }, [value]);

    const onMinus = () => {
        const incrementedNumber = (number || 0) - incrementValue;
        const roundedNumber = roundToOneDecimalPlace(incrementedNumber);
        const normalizedNumber = normalizeBetween(roundedNumber, minValue, maxValue);
        setNumber(normalizedNumber);
    };

    const onPlus = () => {
        const incrementedNumber = (number || 0) + incrementValue;
        const roundedNumber = roundToOneDecimalPlace(incrementedNumber);
        const normalizedNumber = normalizeBetween(roundedNumber, minValue, maxValue);
        setNumber(normalizedNumber);
    };

    return (
        <>
            <SimpleSelect
                value={selectedCondition}
                onChange={(val) => setSelectedCondition(ensureString(val))}
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
                value={number}
                min={0}
                max={10}
                onChange={(event: React.FormEvent<HTMLInputElement>) => {
                    const { value: newNumber } = event.target as HTMLInputElement;
                    setNumber(Number(newNumber));
                }}
                onBlur={(event: React.FormEvent<HTMLInputElement>) => {
                    const target = event.target as HTMLInputElement;
                    const normalizedNumber = normalizeBetween(
                        Number.isNaN(+target.value)
                            ? 0
                            : roundToOneDecimalPlace(Number(target.value)),
                        minValue,
                        maxValue
                    );
                    setNumber(normalizedNumber);
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
                    const newValue = conditionNumberToValue(selectedCondition, number);
                    onSearch(newValue);
                }}
            >
                <ArrowRightIcon />
            </Button>
        </>
    );
}

export default ConditionNumber;

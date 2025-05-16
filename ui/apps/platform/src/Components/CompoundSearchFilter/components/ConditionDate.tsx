import React from 'react';
import { Button, DatePicker, Flex, SelectOption } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import { format } from 'date-fns';

import { ensureString } from 'utils/ensure';
import { dateConditions } from '../utils/utils';

import SimpleSelect from './SimpleSelect';

export type ConditionDate = { condition: string; date: string };

export type ConditionDateProps = {
    value: ConditionDate;
    onChange: (value: ConditionDate) => void;
    onSearch: (value: ConditionDate) => void;
};

function dateFormat(date: Date): string {
    return format(date, 'MM/DD/YYYY');
}

function dateParse(date: string): Date {
    const split = date.split('/');
    if (split.length !== 3) {
        return new Date('Invalid Date');
    }
    const month = split[0];
    const day = split[1];
    const year = split[2];
    if (month.length !== 2 || day.length !== 2 || year.length !== 4) {
        return new Date('Invalid Date');
    }
    return new Date(
        `${year.padStart(4, '0')}-${month.padStart(2, '0')}-${day.padStart(2, '0')}T00:00:00`
    );
}

function ConditionDate({ value, onChange, onSearch }: ConditionDateProps) {
    return (
        <Flex spaceItems={{ default: 'spaceItemsNone' }}>
            <SimpleSelect
                value={value.condition}
                onChange={(val) =>
                    onChange({
                        ...value,
                        condition: ensureString(val),
                    })
                }
                ariaLabelMenu="Condition selector menu"
                ariaLabelToggle="Condition selector toggle"
            >
                {dateConditions.map((condition) => {
                    return (
                        <SelectOption key={condition} value={condition}>
                            {condition}
                        </SelectOption>
                    );
                })}
            </SimpleSelect>
            <DatePicker
                aria-label="Filter by date"
                buttonAriaLabel="Filter by date toggle"
                value={value.date}
                onChange={(_, newValue) => {
                    onChange({ ...value, date: newValue });
                }}
                dateFormat={dateFormat}
                dateParse={dateParse}
                placeholder="MM/DD/YYYY"
                invalidFormatText="Enter valid date: MM/DD/YYYY"
            />
            <Button
                variant="control"
                aria-label="Apply condition and date input to search"
                onClick={() => {
                    const date = dateParse(value.date);
                    if (!Number.isNaN(date.getTime())) {
                        onSearch(value);
                        onChange({ ...value, date: '' });
                    }
                }}
            >
                <ArrowRightIcon />
            </Button>
        </Flex>
    );
}

export default ConditionDate;

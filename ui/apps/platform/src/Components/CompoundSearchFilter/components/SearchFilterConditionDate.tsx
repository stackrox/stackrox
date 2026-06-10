import { useState } from 'react';
import { Button, DatePicker, SelectOption, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import { format } from 'date-fns';

import {
    dateConditionMap,
    dateConditions,
    dateRangeCondition,
    serializeAbsoluteDateRange,
} from '../utils/utils';
import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';

import SimpleSelect from './SimpleSelect';

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

function isValidDate(date: Date): boolean {
    return !Number.isNaN(date.getTime());
}

type DateCondition = (typeof dateConditions)[number] | typeof dateRangeCondition;

export type SearchFilterConditionDateProps = {
    attribute: GenericSearchFilterAttribute;
    isBetweenEnabled?: boolean;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function SearchFilterConditionDate({
    attribute,
    isBetweenEnabled = false,
    isDisabled = false,
    onSearch,
}: SearchFilterConditionDateProps) {
    const { searchTerm: category } = attribute;

    const [conditionExternal, setConditionExternal] = useState<DateCondition>(dateConditions[1]);
    const [dateString, setDateString] = useState('');
    const [startDateString, setStartDateString] = useState('');
    const [endDateString, setEndDateString] = useState('');

    const conditions: DateCondition[] = isBetweenEnabled
        ? [...dateConditions, dateRangeCondition]
        : [...dateConditions];

    const startDate = dateParse(startDateString);

    function onStartDateChange(value: string) {
        setStartDateString(value);
        const datePicked = dateParse(value);
        if (isValidDate(datePicked)) {
            // Default the end date to the day after the start date (PatternFly date-range pattern).
            const dayAfterStart = new Date(datePicked);
            dayAfterStart.setDate(dayAfterStart.getDate() + 1);
            setEndDateString(dateFormat(dayAfterStart));
        } else {
            setEndDateString('');
        }
    }

    function endDateValidator(date: Date): string {
        if (!isValidDate(startDate)) {
            return '';
        }
        return date.getTime() >= startDate.getTime()
            ? ''
            : 'The end date must be on or after the start date';
    }

    function onApply() {
        if (conditionExternal === dateRangeCondition) {
            const start = dateParse(startDateString);
            const end = dateParse(endDateString);
            if (isValidDate(start) && isValidDate(end) && start.getTime() <= end.getTime()) {
                onSearch([
                    {
                        action: 'APPEND',
                        category,
                        value: serializeAbsoluteDateRange(start.getTime(), end.getTime()),
                    },
                ]);
                setStartDateString('');
                setEndDateString('');
            }
            return;
        }

        const dateConditionInternal = dateConditionMap[conditionExternal];
        const date = dateParse(dateString);
        if (dateConditionInternal && !Number.isNaN(date.getTime())) {
            onSearch([
                {
                    action: 'APPEND',
                    category,
                    value: `${dateConditionInternal}${dateString}`,
                },
            ]);
            setDateString('');
        }
    }

    return (
        <>
            <ToolbarItem>
                <SimpleSelect
                    isDisabled={isDisabled}
                    value={conditionExternal}
                    onChange={(conditionSelected) =>
                        setConditionExternal(conditionSelected as DateCondition)
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
            {conditionExternal === dateRangeCondition ? (
                <>
                    <ToolbarItem>
                        <DatePicker
                            aria-label="Filter by start date"
                            buttonAriaLabel="Filter by start date toggle"
                            isDisabled={isDisabled}
                            value={startDateString}
                            onChange={(_, datePicked) => {
                                onStartDateChange(datePicked);
                            }}
                            dateFormat={dateFormat}
                            dateParse={dateParse}
                            placeholder="MM/DD/YYYY"
                            invalidFormatText="Enter valid date: MM/DD/YYYY"
                        />
                    </ToolbarItem>
                    <ToolbarItem alignSelf="center">to</ToolbarItem>
                    <ToolbarItem>
                        <DatePicker
                            aria-label="Filter by end date"
                            buttonAriaLabel="Filter by end date toggle"
                            isDisabled={isDisabled || !isValidDate(startDate)}
                            value={endDateString}
                            onChange={(_, datePicked) => {
                                setEndDateString(datePicked);
                            }}
                            rangeStart={isValidDate(startDate) ? startDate : undefined}
                            validators={[endDateValidator]}
                            dateFormat={dateFormat}
                            dateParse={dateParse}
                            placeholder="MM/DD/YYYY"
                            invalidFormatText="Enter valid date: MM/DD/YYYY"
                        />
                    </ToolbarItem>
                </>
            ) : (
                <ToolbarItem>
                    <DatePicker
                        aria-label="Filter by date"
                        buttonAriaLabel="Filter by date toggle"
                        isDisabled={isDisabled}
                        value={dateString}
                        onChange={(_, datePicked) => {
                            setDateString(datePicked);
                        }}
                        dateFormat={dateFormat}
                        dateParse={dateParse}
                        placeholder="MM/DD/YYYY"
                        invalidFormatText="Enter valid date: MM/DD/YYYY"
                    />
                </ToolbarItem>
            )}
            <ToolbarItem>
                <Button
                    icon={<ArrowRightIcon />}
                    variant="control"
                    aria-label="Apply condition and date input to search"
                    onClick={onApply}
                ></Button>
            </ToolbarItem>
        </>
    );
}

export default SearchFilterConditionDate;

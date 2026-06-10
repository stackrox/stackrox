import { useState } from 'react';
import { Button, DatePicker, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { serializeAbsoluteDateRange } from '../utils/utils';
import { dateFormat, dateParse, isValidDate } from '../utils/dateInput';
import type { OnSearchCallback } from '../types';

export type SearchFilterDateRangeProps = {
    category: string;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
};

/**
 * Date-range body for the date-picker filter input (the Between condition).
 * Serializes the picked range to the backend time-range format ("tr/<startMs>-<endMs>").
 */
function SearchFilterDateRange({
    category,
    isDisabled = false,
    onSearch,
}: SearchFilterDateRangeProps) {
    const [startDateString, setStartDateString] = useState('');
    const [endDateString, setEndDateString] = useState('');

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
    }

    return (
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

export default SearchFilterDateRange;

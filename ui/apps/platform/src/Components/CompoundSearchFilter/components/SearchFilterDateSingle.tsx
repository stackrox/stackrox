import { useState } from 'react';
import { Button, DatePicker, ToolbarItem } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';

import { dateFormat, dateParse, isValidDate } from '../utils/dateInput';
import type { OnSearchCallback } from '../types';

export type SearchFilterDateSingleProps = {
    conditionPrefix: string;
    category: string;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
};

/**
 * Single-date body for the date-picker filter input.
 * Serializes the picked date with the condition prefix (for example, ">01/15/2034").
 */
function SearchFilterDateSingle({
    conditionPrefix,
    category,
    isDisabled = false,
    onSearch,
}: SearchFilterDateSingleProps) {
    const [dateString, setDateString] = useState('');

    function onApply() {
        const date = dateParse(dateString);
        if (conditionPrefix && isValidDate(date)) {
            onSearch([
                {
                    action: 'APPEND',
                    category,
                    value: `${conditionPrefix}${dateString}`,
                },
            ]);
            setDateString('');
        }
    }

    return (
        <>
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

export default SearchFilterDateSingle;

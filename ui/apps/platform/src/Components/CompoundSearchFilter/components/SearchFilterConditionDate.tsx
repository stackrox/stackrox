import { useState } from 'react';
import { Button, DatePicker, Flex, SelectOption } from '@patternfly/react-core';
import { ArrowRightIcon } from '@patternfly/react-icons';
import { format } from 'date-fns';

import { dateConditionMap, dateConditions } from '../utils/utils';
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

export type SearchFilterConditionDateProps = {
    attribute: GenericSearchFilterAttribute;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function SearchFilterConditionDate({ attribute, onSearch }: SearchFilterConditionDateProps) {
    const { searchTerm: category } = attribute;

    const [conditionExternal, setConditionExternal] = useState(dateConditions[1]);
    const [dateString, setDateString] = useState('');

    return (
        <Flex spaceItems={{ default: 'spaceItemsNone' }}>
            <SimpleSelect
                value={conditionExternal}
                onChange={(conditionSelected) =>
                    setConditionExternal(conditionSelected as (typeof dateConditions)[number])
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
                value={dateString}
                onChange={(_, datePicked) => {
                    setDateString(datePicked);
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
                }}
            >
                <ArrowRightIcon />
            </Button>
        </Flex>
    );
}

export default SearchFilterConditionDate;

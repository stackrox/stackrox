import { useState } from 'react';
import { SelectOption, ToolbarItem } from '@patternfly/react-core';

import {
    dateConditionMap,
    dateConditions,
    dateRangeCondition,
    dateRelativeOlderThanCondition,
    dateRelativeRangeCondition,
} from '../utils/utils';
import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';

import SearchFilterDateRange from './SearchFilterDateRange';
import SearchFilterDateRelativeOlderThan from './SearchFilterDateRelativeOlderThan';
import SearchFilterDateRelativeRange from './SearchFilterDateRelativeRange';
import SearchFilterDateSingle from './SearchFilterDateSingle';
import SimpleSelect from './SimpleSelect';

type DateCondition =
    | (typeof dateConditions)[number]
    | typeof dateRelativeOlderThanCondition
    | typeof dateRangeCondition
    | typeof dateRelativeRangeCondition;

const conditions: DateCondition[] = [
    ...dateConditions,
    dateRelativeOlderThanCondition,
    dateRangeCondition,
    dateRelativeRangeCondition,
];

export type SearchFilterConditionDateProps = {
    attribute: GenericSearchFilterAttribute;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

function DateConditionInput({
    condition,
    category,
    isDisabled,
    onSearch,
}: {
    condition: DateCondition;
    category: string;
    isDisabled: boolean;
    onSearch: OnSearchCallback;
}) {
    switch (condition) {
        case dateRelativeOlderThanCondition:
            return (
                <SearchFilterDateRelativeOlderThan
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            );
        case dateRangeCondition:
            return (
                <SearchFilterDateRange
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            );
        case dateRelativeRangeCondition:
            return (
                <SearchFilterDateRelativeRange
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            );
        default:
            return (
                <SearchFilterDateSingle
                    conditionPrefix={dateConditionMap[condition]}
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            );
    }
}

function SearchFilterConditionDate({
    attribute,
    isDisabled = false,
    onSearch,
}: SearchFilterConditionDateProps) {
    const { searchTerm: category } = attribute;

    const [conditionExternal, setConditionExternal] = useState<DateCondition>('On');

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
            <DateConditionInput
                condition={conditionExternal}
                category={category}
                isDisabled={isDisabled}
                onSearch={onSearch}
            />
        </>
    );
}

export default SearchFilterConditionDate;

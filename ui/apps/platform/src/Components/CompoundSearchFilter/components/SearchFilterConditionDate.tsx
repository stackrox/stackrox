import { useState } from 'react';
import { SelectOption, ToolbarItem } from '@patternfly/react-core';

import { dateConditionMap, dateConditions, dateRangeCondition } from '../utils/utils';
import type { GenericSearchFilterAttribute, OnSearchCallback } from '../types';

import SearchFilterDateRange from './SearchFilterDateRange';
import SearchFilterDateSingle from './SearchFilterDateSingle';
import SimpleSelect from './SimpleSelect';

type DateCondition = (typeof dateConditions)[number] | typeof dateRangeCondition;

const conditions: DateCondition[] = [...dateConditions, dateRangeCondition];

export type SearchFilterConditionDateProps = {
    attribute: GenericSearchFilterAttribute;
    isDisabled?: boolean;
    onSearch: OnSearchCallback;
    // does not depend on searchFilter
};

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
            {conditionExternal === dateRangeCondition ? (
                <SearchFilterDateRange
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            ) : (
                <SearchFilterDateSingle
                    conditionPrefix={dateConditionMap[conditionExternal]}
                    category={category}
                    isDisabled={isDisabled}
                    onSearch={onSearch}
                />
            )}
        </>
    );
}

export default SearchFilterConditionDate;

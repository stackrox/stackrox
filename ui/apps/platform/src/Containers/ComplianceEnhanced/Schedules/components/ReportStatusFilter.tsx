import React from 'react';
import { SelectList, SelectOption } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';

import { RunState, runStates } from 'services/ReportsService.types';
import CheckboxSelect from 'Components/CheckboxSelect';

/**
 * Ensures that the given search filter value is converted to an array of valid "RunState" values.
 *
 * Example:
 * ensureReportStates(["WAITING", "PREPARING"]);  // returns ["WAITING", "PREPARING"]
 * ensureReportStates("WAITING");                 // returns [] (since input is not an array)
 * ensureReportStates(undefined);                 // returns []
 *
 * @param searchFilterValue - The input value, which can be a string, an array of strings, or undefined.
 * @returns {RunState[]} - If the input is an array of strings, it filters the values that match valid "RunState"s
 *                         and returns them as an array.
 *                         If the input is not an array or undefined, it returns an empty array.
 */
export function ensureReportStates(searchFilterValue: string | string[] | undefined): RunState[] {
    if (Array.isArray(searchFilterValue)) {
        const reportStates = searchFilterValue.filter((value) => runStates[value]) as RunState[];
        return reportStates;
    }
    return [];
}

export type ReportStatesFilterProps = {
    reportStates: RunState[];
    onChange: (checked: boolean, value: RunState) => void;
};

function ReportStatesFilter({ reportStates, onChange }: ReportStatesFilterProps) {
    function onChangeHandler(checked: boolean, value: string) {
        onChange(checked, value as RunState);
    }

    return (
        <CheckboxSelect
            ariaLabelMenu="Filter by report states select menu"
            toggleLabel="Filter by report states"
            toggleIcon={<FilterIcon />}
            selection={reportStates}
            onChange={onChangeHandler}
        >
            <SelectList>
                <SelectOption
                    key={runStates.PREPARING}
                    value={runStates.PREPARING}
                    hasCheckbox
                    isSelected={reportStates.includes(runStates.PREPARING)}
                >
                    Preparing
                </SelectOption>
                <SelectOption
                    key={runStates.WAITING}
                    value={runStates.WAITING}
                    hasCheckbox
                    isSelected={reportStates.includes(runStates.WAITING)}
                >
                    Waiting
                </SelectOption>
                <SelectOption
                    key={runStates.GENERATED}
                    value={runStates.GENERATED}
                    hasCheckbox
                    isSelected={reportStates.includes(runStates.GENERATED)}
                >
                    Download generated
                </SelectOption>
                <SelectOption
                    key={runStates.DELIVERED}
                    value={runStates.DELIVERED}
                    hasCheckbox
                    isSelected={reportStates.includes(runStates.DELIVERED)}
                >
                    Email delivered
                </SelectOption>
                <SelectOption
                    key={runStates.FAILURE}
                    value={runStates.FAILURE}
                    hasCheckbox
                    isSelected={reportStates.includes(runStates.FAILURE)}
                >
                    Error
                </SelectOption>
            </SelectList>
        </CheckboxSelect>
    );
}

export default ReportStatesFilter;

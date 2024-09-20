import React from 'react';
import { SelectList, SelectOption } from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';

import { RunState, runStates } from 'services/ReportsService.types';
import CheckboxSelect from 'Components/CheckboxSelect';

export type ReportStatusFilterProps = {
    selection: RunState[];
    onChange: (checked: boolean, value: RunState) => void;
};

function ReportStatusFilter({ selection, onChange }: ReportStatusFilterProps) {
    function onChangeHandler(checked: boolean, value: string) {
        onChange(checked, value as RunState);
    }

    return (
        <CheckboxSelect
            ariaLabelMenu="Filter by report status select menu"
            toggleLabel="Filter by report status"
            toggleIcon={<FilterIcon />}
            selection={selection}
            onChange={onChangeHandler}
        >
            <SelectList>
                <SelectOption
                    key={runStates.PREPARING}
                    value={runStates.PREPARING}
                    hasCheckbox
                    isSelected={selection.includes(runStates.PREPARING)}
                >
                    Preparing
                </SelectOption>
                <SelectOption
                    key={runStates.WAITING}
                    value={runStates.WAITING}
                    hasCheckbox
                    isSelected={selection.includes(runStates.WAITING)}
                >
                    Waiting
                </SelectOption>
                <SelectOption
                    key={runStates.GENERATED}
                    value={runStates.GENERATED}
                    hasCheckbox
                    isSelected={selection.includes(runStates.GENERATED)}
                >
                    Download generated
                </SelectOption>
                <SelectOption
                    key={runStates.DELIVERED}
                    value={runStates.DELIVERED}
                    hasCheckbox
                    isSelected={selection.includes(runStates.DELIVERED)}
                >
                    Email delivered
                </SelectOption>
                <SelectOption
                    key={runStates.FAILURE}
                    value={runStates.FAILURE}
                    hasCheckbox
                    isSelected={selection.includes(runStates.FAILURE)}
                >
                    Error
                </SelectOption>
            </SelectList>
        </CheckboxSelect>
    );
}

export default ReportStatusFilter;

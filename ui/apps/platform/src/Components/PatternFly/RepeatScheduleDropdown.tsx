import React, { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';

export type RepeatScheduleDropdownProps = {
    fieldId: string;
    value: string;
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
    showNoResultsOption?: boolean;
    includeDailyOption?: boolean;
    onBlur?: React.FocusEventHandler<HTMLTextAreaElement>;
};

function RepeatScheduleDropdown({
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    showNoResultsOption = false,
    includeDailyOption = false,
    onBlur,
}: RepeatScheduleDropdownProps): ReactElement {
    let options = [
        ...(includeDailyOption
            ? [
                  <SelectOption key="daily" value="DAILY">
                      Daily
                  </SelectOption>,
              ]
            : []),
        <SelectOption key="weekly" value="WEEKLY">
            Weekly
        </SelectOption>,
        <SelectOption key="monthly" value="MONTHLY">
            Monthly
        </SelectOption>,
    ];
    if (showNoResultsOption) {
        options = [<SelectOption isNoResultsOption>None</SelectOption>, ...options];
    }
    return (
        <SelectSingle
            id={fieldId}
            value={value}
            handleSelect={handleSelect}
            isDisabled={!isEditable}
            placeholderText="Select frequency"
            menuAppendTo={() => document.body}
            onBlur={onBlur}
        >
            {options}
        </SelectSingle>
    );
}

export default RepeatScheduleDropdown;

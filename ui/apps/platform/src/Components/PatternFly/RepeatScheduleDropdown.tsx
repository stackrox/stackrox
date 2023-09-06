import React, { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';

export type RepeatScheduleDropdownProps = {
    fieldId: string;
    value: string;
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
    showNoResultsOption?: boolean;
};

function RepeatScheduleDropdown({
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    showNoResultsOption = false,
}: RepeatScheduleDropdownProps): ReactElement {
    let options = [
        <SelectOption value="WEEKLY">Weekly</SelectOption>,
        <SelectOption value="MONTHLY">Monthly</SelectOption>,
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
        >
            {options}
        </SelectSingle>
    );
}

export default RepeatScheduleDropdown;

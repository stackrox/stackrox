import React, { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';

export type RepeatScheduleDropdownProps = {
    fieldId: string;
    value: string;
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
};

function RepeatScheduleDropdown({
    fieldId,
    value,
    handleSelect,
    isEditable = true,
}: RepeatScheduleDropdownProps): ReactElement {
    return (
        <SelectSingle
            id={fieldId}
            value={value}
            handleSelect={handleSelect}
            isDisabled={!isEditable}
            placeholderText="Select frequency"
            menuAppendTo={() => document.body}
        >
            <SelectOption isNoResultsOption>None</SelectOption>
            <SelectOption value="WEEKLY">Weekly</SelectOption>
            <SelectOption value="MONTHLY">Monthly</SelectOption>
        </SelectSingle>
    );
}

export default RepeatScheduleDropdown;

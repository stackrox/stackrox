import type { FocusEventHandler, ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle';

export type RepeatScheduleDropdownProps = {
    fieldId: string;
    value: string;
    handleSelect: (id: string, selection: string) => void;
    isEditable?: boolean;
    hasUnsetOption: boolean;
    onBlur?: FocusEventHandler<HTMLDivElement>;
};

function RepeatScheduleDropdown({
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    hasUnsetOption,
    onBlur,
}: RepeatScheduleDropdownProps): ReactElement {
    const options = [
        <SelectOption key="DAILY" value="DAILY">
            Daily
        </SelectOption>,
        <SelectOption key="WEEKLY" value="WEEKLY">
            Weekly
        </SelectOption>,
        <SelectOption key="MONTHLY" value="MONTHLY">
            Monthly
        </SelectOption>,
    ];
    if (hasUnsetOption) {
        options.push(
            <SelectOption key="UNSET" value="UNSET">
                Not scheduled
            </SelectOption>
        );
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

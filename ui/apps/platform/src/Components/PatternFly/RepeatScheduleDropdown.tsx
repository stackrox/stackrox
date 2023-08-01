import React, { ReactElement } from 'react';
import { SelectOption } from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import SelectSingle from 'Components/SelectSingle';

export type RepeatScheduleDropdownProps = {
    label: string;
    fieldId: string;
    value: string;
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
    isRequired?: boolean;
};

function RepeatScheduleDropdown({
    label,
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    isRequired = false,
}: RepeatScheduleDropdownProps): ReactElement {
    return (
        <FormLabelGroup isRequired={isRequired} label={label} fieldId={fieldId} errors={{}}>
            <SelectSingle
                id={fieldId}
                value={value}
                handleSelect={handleSelect}
                isDisabled={!isEditable}
                placeholderText="Select frequency"
            >
                <SelectOption value="WEEKLY">Weekly</SelectOption>
                <SelectOption value="MONTHLY">Monthly</SelectOption>
            </SelectSingle>
        </FormLabelGroup>
    );
}

export default RepeatScheduleDropdown;

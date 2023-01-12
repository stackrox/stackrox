import React, { ReactElement } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useMultiSelect from 'hooks/useMultiSelect';
import { IntervalType } from 'types/report.proto';

export type DayPickerDropdownProps = {
    label: string;
    fieldId: string;
    value: string[];
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
    isRequired?: boolean;
    intervalType: IntervalType;
};

function DayPickerDropdown({
    label,
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    isRequired = false,
    intervalType,
}: DayPickerDropdownProps): ReactElement {
    const selectSafeValue = value.map((item) => item.toString());
    const {
        isOpen: isDaySelectOpen,
        onToggle: onToggleDaySelect,
        onSelect: onSelectDay,
    } = useMultiSelect(handleDaySelect, selectSafeValue, false);

    function handleDaySelect(selection) {
        handleSelect(fieldId, selection);
    }

    const selectOptions =
        intervalType === 'WEEKLY'
            ? [
                  <SelectOption key="monday" value="1">
                      Monday
                  </SelectOption>,
                  <SelectOption key="tuesday" value="2">
                      Tuesday
                  </SelectOption>,
                  <SelectOption key="wednesday" value="3">
                      Wednesday
                  </SelectOption>,
                  <SelectOption key="thursday" value="4">
                      Thursday
                  </SelectOption>,
                  <SelectOption key="friday" value="5">
                      Friday
                  </SelectOption>,
                  <SelectOption key="saturday" value="6">
                      Saturday
                  </SelectOption>,
                  <SelectOption key="sunday" value="0">
                      Sunday
                  </SelectOption>,
              ]
            : [
                  <SelectOption key="first" value="1">
                      The first of the month
                  </SelectOption>,
                  <SelectOption key="middle" value="15">
                      The middle of the month
                  </SelectOption>,
              ];

    return (
        <FormLabelGroup isRequired={isRequired} label={label} fieldId={fieldId} errors={{}}>
            <Select
                variant={SelectVariant.checkbox}
                aria-label="Select one or more days"
                onToggle={onToggleDaySelect}
                onSelect={onSelectDay}
                selections={selectSafeValue}
                isOpen={isDaySelectOpen}
                isDisabled={!isEditable}
                placeholderText={value.length ? 'Selected days' : 'Select days'}
            >
                {selectOptions}
            </Select>
        </FormLabelGroup>
    );
}

export default DayPickerDropdown;

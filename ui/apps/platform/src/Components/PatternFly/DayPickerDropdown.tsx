import React, { ReactElement } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useMultiSelect from 'hooks/useMultiSelect';
import { IntervalType } from 'types/report.proto';

export type DayPickerDropdownProps = {
    fieldId: string;
    value: string[];
    handleSelect: (id, selection) => void;
    isEditable?: boolean;
    intervalType: IntervalType | null;
    onBlur?: React.FocusEventHandler<HTMLTextAreaElement>;
};

export const daysOfWeek = ['0', '1', '2', '3', '4', '5', '6'] as const;
export const daysOfMonth = ['1', '15'] as const;

export type DayOfWeek = (typeof daysOfWeek)[number];
export type DayOfMonth = (typeof daysOfMonth)[number];

export const daysOfWeekMap: Record<DayOfWeek, string> = {
    '0': 'Sunday',
    '1': 'Monday',
    '2': 'Tuesday',
    '3': 'Wednesday',
    '4': 'Thursday',
    '5': 'Friday',
    '6': 'Saturday',
} as const;

export const daysOfMonthMap: Record<DayOfMonth, string> = {
    '1': 'The first of the month',
    '15': 'The middle of the month',
} as const;

function DayPickerDropdown({
    fieldId,
    value,
    handleSelect,
    isEditable = true,
    intervalType,
    onBlur,
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

    let selectOptions: ReactElement[] = [];

    if (intervalType) {
        selectOptions =
            intervalType === 'WEEKLY'
                ? [
                      <SelectOption key="monday" value="1">
                          {daysOfWeekMap[1]}
                      </SelectOption>,
                      <SelectOption key="tuesday" value="2">
                          {daysOfWeekMap[2]}
                      </SelectOption>,
                      <SelectOption key="wednesday" value="3">
                          {daysOfWeekMap[3]}
                      </SelectOption>,
                      <SelectOption key="thursday" value="4">
                          {daysOfWeekMap[4]}
                      </SelectOption>,
                      <SelectOption key="friday" value="5">
                          {daysOfWeekMap[5]}
                      </SelectOption>,
                      <SelectOption key="saturday" value="6">
                          {daysOfWeekMap[6]}
                      </SelectOption>,
                      <SelectOption key="sunday" value="0">
                          {daysOfWeekMap[0]}
                      </SelectOption>,
                  ]
                : [
                      <SelectOption key="first" value="1">
                          {daysOfMonthMap[1]}
                      </SelectOption>,
                      <SelectOption key="middle" value="15">
                          {daysOfMonthMap[15]}
                      </SelectOption>,
                  ];
    }

    return (
        <Select
            variant={SelectVariant.checkbox}
            aria-label="Select one or more days"
            onToggle={onToggleDaySelect}
            onSelect={onSelectDay}
            selections={selectSafeValue}
            isOpen={isDaySelectOpen}
            isDisabled={!isEditable}
            placeholderText={value.length ? 'Selected days' : 'Select days'}
            menuAppendTo={() => document.body}
            onBlur={onBlur}
        >
            {selectOptions}
        </Select>
    );
}

export default DayPickerDropdown;

import type { FormEvent, ReactElement } from 'react';
import { FormSection, TimePicker } from '@patternfly/react-core';
import type { FormikProps } from 'formik';

import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import type {
    DailySchedule,
    MonthlySchedule,
    ScheduleBase,
    WeeklySchedule,
} from 'types/schedule.proto';
import { getHourMinuteStringFromScheduleBase } from 'utils/dateUtils';

import type { DeliveryType } from '../reports.types';

export type ScheduleFormSectionProps<T extends DeliveryType = DeliveryType> = {
    formik: FormikProps<T>;
};

function ScheduleFormSection<T extends DeliveryType = DeliveryType>({
    formik,
}: ScheduleFormSectionProps<T>): ReactElement {
    function handleSelectIntervalType(id: string, intervalType: string): void {
        const scheduleBase: ScheduleBase = {
            hour: formik.values.schedule?.hour ?? 0,
            minute: formik.values.schedule?.minute ?? 0,
        };

        switch (intervalType) {
            case 'DAILY': {
                const schedule: DailySchedule = { ...scheduleBase, intervalType };
                formik.setFieldValue('schedule', schedule);
                break;
            }
            case 'WEEKLY': {
                const schedule: WeeklySchedule = {
                    ...scheduleBase,
                    intervalType,
                    daysOfWeek: { days: [] },
                };
                formik.setFieldValue('schedule', schedule);
                break;
            }
            case 'MONTHLY': {
                const schedule: MonthlySchedule = {
                    ...scheduleBase,
                    intervalType,
                    daysOfMonth: { days: [] },
                };
                formik.setFieldValue('schedule', schedule);
                break;
            }
            default:
                break;
        }
    }

    function handleSelectDays(id: string, selection: string[]) {
        formik.setFieldValue(
            id,
            selection.map((day) => Number(day)),
            true
        );
    }

    function onChangeTime(
        _event: FormEvent<HTMLInputElement>,
        _time: string,
        hour: number | null | undefined,
        minute: number | null | undefined
    ): void {
        // Arguments have null value if incorrect. Do not replace previous values.
        if (typeof hour === 'number' && typeof minute === 'number') {
            formik.setFieldValue('schedule.hour', hour);
            formik.setFieldValue('schedule.minute', minute);
        }
    }

    const fieldIdForDays =
        formik.values.schedule?.intervalType === 'WEEKLY'
            ? 'schedule.daysOfWeek.days'
            : 'schedule.daysOfMonth.days';

    return (
        <FormSection title="Schedule" titleElement="h3">
            <FormLabelGroup
                label="Frequency"
                fieldId="schedule.intervalType"
                isRequired
                errors={formik.errors}
                touched={formik.touched}
            >
                <RepeatScheduleDropdown
                    fieldId="schedule.intervalType"
                    value={formik.values.schedule?.intervalType ?? ''}
                    handleSelect={handleSelectIntervalType}
                    includeDailyOption
                    onBlur={formik.handleBlur}
                />
            </FormLabelGroup>
            <FormLabelGroup
                label="Days"
                fieldId={fieldIdForDays}
                errors={formik.errors}
                isRequired={
                    formik.values.schedule?.intervalType === 'WEEKLY' ||
                    formik.values.schedule?.intervalType === 'MONTHLY'
                }
                touched={formik.touched}
            >
                {formik.values.schedule?.intervalType === 'WEEKLY' ||
                formik.values.schedule?.intervalType === 'MONTHLY' ? (
                    <DayPickerDropdown
                        fieldId={fieldIdForDays}
                        value={
                            formik.values.schedule.intervalType === 'WEEKLY'
                                ? formik.values.schedule.daysOfWeek.days.map((day) => String(day))
                                : formik.values.schedule.daysOfMonth.days.map((day) => String(day))
                        }
                        handleSelect={handleSelectDays}
                        intervalType={formik.values.schedule.intervalType}
                        toggleId={fieldIdForDays}
                        onBlur={() => {
                            formik.setFieldTouched(fieldIdForDays, true);
                        }}
                    />
                ) : (
                    <>Not applicable</>
                )}
            </FormLabelGroup>
            <FormLabelGroup
                label="Time"
                fieldId="time"
                errors={formik.errors}
                isRequired
                touched={formik.touched}
                helperText="Select or enter time between 00:00 and 23:59 UTC"
            >
                <TimePicker
                    time={
                        formik.values.schedule
                            ? getHourMinuteStringFromScheduleBase(formik.values.schedule)
                            : '00:00'
                    }
                    is24Hour
                    onChange={onChangeTime}
                    inputProps={{
                        onBlur: formik.handleBlur,
                        // name: 'parameters.time',
                    }}
                    invalidFormatErrorMessage="" // error messaging is handled by FormLabelGroup
                    invalidMinMaxErrorMessage=""
                />
            </FormLabelGroup>
        </FormSection>
    );
}

export default ScheduleFormSection;

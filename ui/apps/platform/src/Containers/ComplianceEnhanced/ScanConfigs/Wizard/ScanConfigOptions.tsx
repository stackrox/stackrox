import React, { ReactElement } from 'react';
import { FormikProps } from 'formik';
import {
    Divider,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Stack,
    StackItem,
    TextArea,
    TextInput,
    TimePicker,
    Title,
} from '@patternfly/react-core';

import DayPickerDropdown from 'Components/PatternFly/DayPickerDropdown';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import RepeatScheduleDropdown from 'Components/PatternFly/RepeatScheduleDropdown';
import { getTimeHoursMinutes } from 'utils/dateUtils';

import { ScanConfigFormValues } from './useFormikScanConfig';

export type ScanConfigOptionsProps = {
    formik: FormikProps<ScanConfigFormValues>;
};

function ScanConfigOptions({ formik }: ScanConfigOptionsProps): ReactElement {
    function handleSelectChange(id: string, value: string): void {
        formik.setFieldValue('parameters.daysOfWeek', []);
        formik.setFieldValue('parameters.daysOfMonth', []);
        formik.setFieldValue(id, value);
    }

    function handleTimeChange(
        _event: React.FormEvent<HTMLInputElement>,
        time: string,
        hour?: number,
        minute?: number,
        _seconds?: number,
        isValid?: boolean
    ): void {
        formik.setFieldTouched('parameters.time', true, true);
        if (isValid && hour !== undefined) {
            const date = new Date();
            date.setHours(hour, minute, 0, 0);
            const timeString = getTimeHoursMinutes(date);
            formik.setFieldValue('parameters.time', timeString);
        } else {
            formik.setFieldValue('parameters.time', time);
        }
    }

    function onScheduledDaysChange(id: string, selection: string[]) {
        formik.setFieldValue(id, selection);
    }

    function updateFormikTouched(fieldName: string) {
        formik.setFieldTouched(fieldName, true, true);
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Configuration options</Title>
                    </FlexItem>
                    <FlexItem>Set up name, schedule, and options</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-u-py-lg pf-u-px-lg">
                <Stack hasGutter>
                    <StackItem>
                        <Stack hasGutter>
                            <StackItem>
                                <FormLabelGroup
                                    label="Name"
                                    isRequired
                                    fieldId="parameters.name"
                                    errors={formik.errors}
                                    touched={formik.touched}
                                >
                                    <TextInput
                                        isRequired
                                        type="text"
                                        id="parameters.name"
                                        name="parameters.name"
                                        value={formik.values.parameters.name}
                                        onChange={(_value, event) => formik.handleChange(event)}
                                        onBlur={formik.handleBlur}
                                    />
                                </FormLabelGroup>
                            </StackItem>
                            <StackItem>
                                <FormLabelGroup
                                    label="Description"
                                    fieldId="parameters.description"
                                    errors={formik.errors}
                                >
                                    <TextArea
                                        isRequired
                                        type="text"
                                        id="parameters.description"
                                        name="parameters.description"
                                        value={formik.values.parameters.description}
                                        onChange={(_value, event) => formik.handleChange(event)}
                                        onBlur={formik.handleBlur}
                                    />
                                </FormLabelGroup>
                            </StackItem>
                        </Stack>
                    </StackItem>
                    <StackItem>
                        <Divider component="div" />
                    </StackItem>
                    <StackItem>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h3">Configure schedule</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure or setup a schedule to scan on a recurring basis.
                            </FlexItem>
                            <FlexItem flex={{ default: 'flexNone' }}>
                                <Flex direction={{ default: 'column' }}>
                                    <Flex direction={{ default: 'row' }}>
                                        <FlexItem>
                                            <FormLabelGroup
                                                label="Frequency"
                                                fieldId="parameters.intervalType"
                                                isRequired
                                                errors={formik.errors}
                                                touched={formik.touched}
                                            >
                                                <RepeatScheduleDropdown
                                                    fieldId="parameters.intervalType"
                                                    value={
                                                        formik.values.parameters.intervalType || ''
                                                    }
                                                    handleSelect={handleSelectChange}
                                                    includeDailyOption
                                                    onBlur={() =>
                                                        updateFormikTouched(
                                                            'parameters.intervalType'
                                                        )
                                                    }
                                                />
                                            </FormLabelGroup>
                                        </FlexItem>
                                        <FlexItem>
                                            <FormLabelGroup
                                                label="On day(s)"
                                                fieldId={
                                                    formik.values.parameters.intervalType ===
                                                    'WEEKLY'
                                                        ? 'parameters.daysOfWeek'
                                                        : 'parameters.daysOfMonth'
                                                }
                                                errors={formik.errors}
                                                isRequired={
                                                    formik.values.parameters.intervalType ===
                                                        'WEEKLY' ||
                                                    formik.values.parameters.intervalType ===
                                                        'MONTHLY'
                                                }
                                                touched={formik.touched}
                                            >
                                                <DayPickerDropdown
                                                    fieldId={
                                                        formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? 'parameters.daysOfWeek'
                                                            : 'parameters.daysOfMonth'
                                                    }
                                                    value={
                                                        formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? formik.values.parameters.daysOfWeek ||
                                                              []
                                                            : formik.values.parameters
                                                                  .daysOfMonth || []
                                                    }
                                                    handleSelect={onScheduledDaysChange}
                                                    intervalType={
                                                        formik.values.parameters.intervalType
                                                    }
                                                    isEditable={
                                                        formik.values.parameters.intervalType ===
                                                            'MONTHLY' ||
                                                        formik.values.parameters.intervalType ===
                                                            'WEEKLY'
                                                    }
                                                    onBlur={() => {
                                                        const fieldName =
                                                            formik.values.parameters
                                                                .intervalType === 'WEEKLY'
                                                                ? 'parameters.daysOfWeek'
                                                                : 'parameters.daysOfMonth';

                                                        updateFormikTouched(fieldName);
                                                    }}
                                                />
                                            </FormLabelGroup>
                                        </FlexItem>
                                    </Flex>
                                    <FlexItem>
                                        <FormLabelGroup
                                            label="Time"
                                            fieldId="parameters.time"
                                            errors={formik.errors}
                                            isRequired
                                            touched={formik.touched}
                                        >
                                            <TimePicker
                                                time={formik.values.parameters.time}
                                                onChange={handleTimeChange}
                                            />
                                        </FormLabelGroup>
                                    </FlexItem>
                                </Flex>
                            </FlexItem>
                        </Flex>
                    </StackItem>
                </Stack>
            </Form>
        </>
    );
}

export default ScanConfigOptions;

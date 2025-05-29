import React, { ReactElement } from 'react';
import { FormikContextType, useFormikContext } from 'formik';
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

import usePageAction from 'hooks/usePageAction';
import { PageActions, ScanConfigFormValues } from '../compliance.scanConfigs.utils';

import { helperTextForName, helperTextForNameEdit, helperTextForTime } from './useFormikScanConfig';

import './ScanConfigOptions.css';

function ScanConfigOptions(): ReactElement {
    const formik: FormikContextType<ScanConfigFormValues> = useFormikContext();
    const { pageAction } = usePageAction<PageActions>();
    const isEditAction = pageAction === 'edit';

    function handleSelectChange(id: string, value: string): void {
        formik.setFieldValue('parameters.daysOfWeek', []);
        formik.setFieldValue('parameters.daysOfMonth', []);
        formik.setFieldValue(id, value);
    }

    function handleTimeChange(_event: React.FormEvent<HTMLInputElement>, time: string): void {
        formik.setFieldValue('parameters.time', time);
    }

    function onScheduledDaysChange(id: string, selection: string[]) {
        formik.setFieldValue(id, selection);
    }

    return (
        <>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h2">Parameters</Title>
                    </FlexItem>
                    <FlexItem>Set name and schedule to scan on a recurring basis</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <Form className="pf-v5-u-py-lg pf-v5-u-px-lg" id="scan-schedules-parameters">
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
                                    helperText={
                                        isEditAction ? helperTextForNameEdit : helperTextForName
                                    }
                                >
                                    <TextInput
                                        isRequired
                                        type="text"
                                        id="parameters.name"
                                        name="parameters.name"
                                        value={formik.values.parameters.name}
                                        isDisabled={isEditAction}
                                        validated={
                                            formik.errors?.parameters?.name &&
                                            formik.touched?.parameters?.name
                                                ? 'error'
                                                : 'default'
                                        }
                                        onChange={(event) => formik.handleChange(event)}
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
                                        onChange={(event) => formik.handleChange(event)}
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
                                <Title headingLevel="h3">Schedule</Title>
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
                                                    onBlur={formik.handleBlur}
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
                                                    toggleId={
                                                        formik.values.parameters.intervalType ===
                                                        'WEEKLY'
                                                            ? 'parameters.daysOfWeek'
                                                            : 'parameters.daysOfMonth'
                                                    }
                                                    onBlur={formik.handleBlur}
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
                                            helperText={helperTextForTime}
                                        >
                                            <TimePicker
                                                time={formik.values.parameters.time}
                                                is24Hour
                                                onChange={handleTimeChange}
                                                inputProps={{
                                                    onBlur: formik.handleBlur,
                                                    name: 'parameters.time',
                                                }}
                                                invalidFormatErrorMessage="" // error messaging is handled by FormLabelGroup
                                                invalidMinMaxErrorMessage=""
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

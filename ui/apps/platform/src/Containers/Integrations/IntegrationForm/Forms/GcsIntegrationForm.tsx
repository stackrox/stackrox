import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, FormSelect, Switch } from '@patternfly/react-core';
import * as yup from 'yup';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import ScheduleIntervalOptions from '../FormSchedule/ScheduleIntervalOptions';
import ScheduleWeeklyOptions from '../FormSchedule/ScheduleWeeklyOptions';
import ScheduleDailyOptions from '../FormSchedule/ScheduleDailyOptions';

export type GcsIntegration = {
    id?: string;
    name: string;
    backupsToKeep: number;
    schedule: {
        intervalType: 'UNSET' | 'DAILY' | 'WEEKLY';
        weekly?: {
            day: number;
        };
        hour: number;
        minute: number;
    };
    gcs: {
        bucket: string;
        objectPrefix: string;
        useWorkloadId: boolean;
        serviceAccount: string;
    };
    type: 'gcs';
    enabled: boolean;
    categories: string[];
};

export type GcsIntegrationFormValues = {
    externalBackup: GcsIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    externalBackup: yup.object().shape({
        name: yup.string().required('Required'),
        backupsToKeep: yup.number().required('Required'),
        schedule: yup.object().shape({
            intervalType: yup.string().required('Required'),
            weekly: yup.object().shape({
                day: yup.number(),
            }),
            hour: yup.number().required('Required'),
            minute: yup.number().required('Required'),
        }),
        gcs: yup.object().shape({
            bucket: yup.string().required('Required'),
            objectPrefix: yup.string(),
            useWorkloadId: yup.bool(),
            serviceAccount: yup.string(),
        }),
        type: yup.string().matches(/gcs/),
        enabled: yup.bool(),
        categories: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: GcsIntegrationFormValues = {
    externalBackup: {
        name: '',
        backupsToKeep: 0,
        schedule: {
            intervalType: 'DAILY',
            hour: 0,
            minute: 0,
        },
        gcs: {
            bucket: '',
            objectPrefix: '',
            useWorkloadId: false,
            serviceAccount: '',
        },
        type: 'gcs',
        enabled: true,
        categories: [],
    },
    updatePassword: true,
};

function GcsIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GcsIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.externalBackup = {
            ...formInitialValues.externalBackup,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.externalBackup.gcs.serviceAccount = '';
    }
    const {
        values,
        errors,
        setFieldValue,
        isSubmitting,
        isTesting,
        onSave,
        onTest,
        onCancel,
        message,
    } = useIntegrationForm<GcsIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup
                        isRequired
                        label="Name"
                        fieldId="externalBackup.name"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.name"
                            name="externalBackup.name"
                            value={values.externalBackup.name}
                            placeholder="(ex. Google Cloud Storage)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Backups to Retain"
                        fieldId="externalBackup.backupsToKeep"
                        errors={errors}
                    >
                        <TextInput
                            type="number"
                            id="externalBackup.backupsToKeep"
                            name="externalBackup.backupsToKeep"
                            value={values.externalBackup.backupsToKeep}
                            placeholder="(ex. 5)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Schedule Interval"
                        fieldId="externalBackup.schedule.intervalType"
                        errors={errors}
                    >
                        <FormSelect
                            id="externalBackup.schedule.intervalType"
                            value={values.externalBackup.schedule.intervalType}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        >
                            <ScheduleIntervalOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    {values.externalBackup.schedule.intervalType === 'WEEKLY' && (
                        <FormLabelGroup
                            isRequired
                            label="Schedule Day of Week"
                            fieldId="externalBackup.schedule.weekly.day"
                            errors={errors}
                        >
                            <FormSelect
                                id="externalBackup.schedule.weekly.day"
                                value={values.externalBackup.schedule?.weekly?.day}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            >
                                <ScheduleWeeklyOptions />
                            </FormSelect>
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired
                        label="Schedule Time of Day"
                        fieldId="externalBackup.schedule.hour"
                        errors={errors}
                    >
                        <FormSelect
                            id="externalBackup.schedule.hour"
                            value={values.externalBackup.schedule.hour}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        >
                            <ScheduleDailyOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Bucket"
                        fieldId="externalBackup.gcs.bucket"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.gcs.bucket"
                            name="externalBackup.gcs.bucket"
                            value={values.externalBackup.gcs.bucket}
                            placeholder="(ex. stackrox.backups)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Object Prefix"
                        fieldId="externalBackup.gcs.objectPrefix"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.gcs.objectPrefix"
                            name="externalBackup.gcs.objectPrefix"
                            value={values.externalBackup.gcs.objectPrefix}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use Workload Identity"
                        fieldId="externalBackup.gcs.useWorkloadId"
                        errors={errors}
                    >
                        <Switch
                            id="externalBackup.gcs.useWorkloadId"
                            name="externalBackup.gcs.useWorkloadId"
                            aria-label="use container iam role"
                            isChecked={values.externalBackup.gcs.useWorkloadId}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && !values.externalBackup.gcs.useWorkloadId && (
                        <FormLabelGroup
                            label="Update Password"
                            fieldId="updatePassword"
                            helperText="Setting this to false will use the currently stored credentials, if they exist."
                            errors={errors}
                        >
                            <Switch
                                id="updatePassword"
                                name="updatePassword"
                                aria-label="update password"
                                isChecked={values.updatePassword}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    {values.updatePassword && !values.externalBackup.gcs.useWorkloadId && (
                        <FormLabelGroup
                            label="Service Account (JSON)"
                            fieldId="externalBackup.gcs.serviceAccount"
                            isRequired
                            errors={errors}
                        >
                            <TextInput
                                type="password"
                                id="externalBackup.gcs.serviceAccount"
                                name="externalBackup.gcs.serviceAccount"
                                value={values.externalBackup.gcs.serviceAccount}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={onTest}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                    >
                        Test
                    </FormTestButton>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default GcsIntegrationForm;

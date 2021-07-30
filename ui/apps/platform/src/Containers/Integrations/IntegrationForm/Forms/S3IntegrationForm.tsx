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
import FormMessageBanner from '../FormMessageBanner';
import FormLabelGroup from '../FormLabelGroup';
import ScheduleIntervalOptions from '../FormSchedule/ScheduleIntervalOptions';
import ScheduleWeeklyOptions from '../FormSchedule/ScheduleWeeklyOptions';
import ScheduleDailyOptions from '../FormSchedule/ScheduleDailyOptions';

export type S3Integration = {
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
    s3: {
        bucket: string;
        objectPrefix: string;
        endpoint: string;
        region: string;
        useIam: boolean;
        accessKeyId: string;
        secretAccessKey: string;
    };
    type: 's3';
    enabled: boolean;
    categories: string[];
};

export type S3IntegrationFormValues = {
    externalBackup: S3Integration;
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
        s3: yup.object().shape({
            bucket: yup.string().required('Required'),
            objectPrefix: yup.string().required('Required'),
            endpoint: yup.string().required('Required'),
            region: yup.string().required('Required'),
            useIam: yup.bool(),
            accessKeyId: yup.string(),
            secretAccessKey: yup.string(),
        }),
        type: yup.string().matches(/s3/),
        enabled: yup.bool(),
        categories: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: S3IntegrationFormValues = {
    externalBackup: {
        name: '',
        backupsToKeep: 0,
        schedule: {
            intervalType: 'DAILY',
            hour: 0,
            minute: 0,
        },
        s3: {
            bucket: '',
            objectPrefix: '',
            endpoint: '',
            region: '',
            useIam: false,
            accessKeyId: '',
            secretAccessKey: '',
        },
        type: 's3',
        enabled: true,
        categories: [],
    },
    updatePassword: true,
};

function S3IntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<S3Integration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.externalBackup = {
            ...formInitialValues.externalBackup,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.externalBackup.s3.accessKeyId = '';
        formInitialValues.externalBackup.s3.secretAccessKey = '';
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
    } = useIntegrationForm<S3IntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            {message && <FormMessageBanner message={message} />}
            <PageSection variant="light" isFilled hasOverflowScroll>
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
                            placeholder="(ex. Amazon S3)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Backups to Retain"
                        fieldId="externalBackup.backupsToKeep"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
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
                        fieldId="externalBackup.s3.bucket"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.bucket"
                            name="externalBackup.s3.bucket"
                            value={values.externalBackup.s3.bucket}
                            placeholder="(ex. stackrox.backups)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Object Prefix"
                        fieldId="externalBackup.s3.objectPrefix"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.objectPrefix"
                            name="externalBackup.s3.objectPrefix"
                            value={values.externalBackup.s3.objectPrefix}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        fieldId="externalBackup.s3.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.endpoint"
                            name="externalBackup.s3.endpoint"
                            value={values.externalBackup.s3.endpoint}
                            placeholder="(ex. s3.us-west-2.amazonaws.com)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Region"
                        fieldId="externalBackup.s3.region"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.region"
                            name="externalBackup.s3.region"
                            value={values.externalBackup.s3.region}
                            placeholder="(ex. us-west-2)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use Container IAM Role"
                        fieldId="externalBackup.s3.useIam"
                        errors={errors}
                    >
                        <Switch
                            id="externalBackup.s3.useIam"
                            name="externalBackup.s3.useIam"
                            aria-label="use container iam role"
                            isChecked={values.externalBackup.s3.useIam}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && (
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
                    {values.updatePassword && !values.externalBackup.s3.useIam && (
                        <>
                            <FormLabelGroup
                                label="Access Key ID"
                                fieldId="externalBackup.s3.accessKeyId"
                                isRequired
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="externalBackup.s3.accessKeyId"
                                    name="externalBackup.s3.accessKeyId"
                                    value={values.externalBackup.s3.accessKeyId}
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Secret Access Key"
                                fieldId="externalBackup.s3.secretAccessKey"
                                isRequired
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="externalBackup.s3.secretAccessKey"
                                    name="externalBackup.s3.secretAccessKey"
                                    value={values.externalBackup.s3.secretAccessKey}
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </>
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

export default S3IntegrationForm;

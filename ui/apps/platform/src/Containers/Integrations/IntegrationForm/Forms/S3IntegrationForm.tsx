/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import { Checkbox, Form, FormSelect, PageSection, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';

import { BackupIntegrationBase } from 'services/BackupIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import ScheduleIntervalOptions from '../FormSchedule/ScheduleIntervalOptions';
import ScheduleWeeklyOptions from '../FormSchedule/ScheduleWeeklyOptions';
import ScheduleDailyOptions from '../FormSchedule/ScheduleDailyOptions';

export type S3Integration = {
    s3: {
        bucket: string;
        objectPrefix: string;
        endpoint: string;
        region: string;
        useIam: boolean;
        accessKeyId: string;
        secretAccessKey: string;
        usePathstyle: boolean;
    };
    type: 's3';
} & BackupIntegrationBase;

export type S3IntegrationFormValues = {
    externalBackup: S3Integration;
    updatePassword: boolean;
};

function requireCredentials(value, context: yup.TestContext) {
    const requirePasswordField =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        context?.from[2]?.value?.updatePassword || false;
    const useIam = context?.parent?.useIam;

    if (!requirePasswordField || useIam) {
        return true;
    }

    const trimmedValue = value?.trim();
    return !!trimmedValue;
}

export const validationSchema = yup.object().shape({
    externalBackup: yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        backupsToKeep: yup
            .number()
            .required('Number of backups to keep is required')
            .min(1, 'Number of backups to keep must be 1 or greater'),
        schedule: yup.object().shape({
            intervalType: yup.string().trim().required('Interval is required'),
            weekly: yup.object().shape({
                day: yup.number(),
            }),
            hour: yup.number(),
            minute: yup.number(),
        }),
        s3: yup.object().shape({
            bucket: yup.string().trim().required('Bucket is required'),
            objectPrefix: yup.string(),
            endpoint: yup.string(),
            region: yup.string().trim().required('Region is required'),
            useIam: yup.bool(),
            accessKeyId: yup
                .string()
                .trim()
                .test('accessKeyId-test', 'An access key ID is required', requireCredentials),
            secretAccessKey: yup
                .string()
                .trim()
                .test(
                    'secretAccessKey-test',
                    'A secret access key is required',
                    requireCredentials
                ),
        }),
        type: yup.string().matches(/s3/),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: S3IntegrationFormValues = {
    externalBackup: {
        id: '',
        name: '',
        backupsToKeep: 1,
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
    },
    updatePassword: true,
};

function S3IntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<S3Integration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };

    if (initialValues) {
        formInitialValues.externalBackup = {
            ...formInitialValues.externalBackup,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.externalBackup.s3.accessKeyId = '';
        formInitialValues.externalBackup.s3.secretAccessKey = '';

        // Don't assume user wants to change password; that has caused confusing UX.
        formInitialValues.updatePassword = false;
    }
    const {
        values,
        touched,
        errors,
        dirty,
        isValid,
        setFieldValue,
        handleBlur,
        isSubmitting,
        isTesting,
        onSave,
        onTest,
        onCancel,
        message,
    } = useIntegrationForm<S3IntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function updateKeysOnChange(value, event) {
        void setFieldValue(event.target.id, value);
        if (value === true) {
            void setFieldValue('externalBackup.s3.accessKeyId', '');
            void setFieldValue('externalBackup.s3.secretAccessKey', '');
        }
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('externalBackup.s3.accessKeyId', '');
        setFieldValue('externalBackup.s3.secretAccessKey', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        isRequired
                        label="Integration name"
                        fieldId="externalBackup.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="externalBackup.name"
                            value={values.externalBackup.name}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Backups to retain"
                        fieldId="externalBackup.backupsToKeep"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="externalBackup.backupsToKeep"
                            value={values.externalBackup.backupsToKeep}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Schedule interval"
                        fieldId="externalBackup.schedule.intervalType"
                        touched={touched}
                        errors={errors}
                    >
                        <FormSelect
                            id="externalBackup.schedule.intervalType"
                            value={values.externalBackup.schedule.intervalType}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            <ScheduleIntervalOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    {values.externalBackup.schedule.intervalType === 'WEEKLY' && (
                        <FormLabelGroup
                            isRequired
                            label="Schedule day of week"
                            fieldId="externalBackup.schedule.weekly.day"
                            touched={touched}
                            errors={errors}
                        >
                            <FormSelect
                                id="externalBackup.schedule.weekly.day"
                                value={values.externalBackup.schedule?.weekly?.day}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            >
                                <ScheduleWeeklyOptions />
                            </FormSelect>
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired
                        label="Schedule time of day"
                        fieldId="externalBackup.schedule.hour"
                        touched={touched}
                        errors={errors}
                    >
                        <FormSelect
                            id="externalBackup.schedule.hour"
                            value={values.externalBackup.schedule.hour}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            <ScheduleDailyOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Bucket"
                        fieldId="externalBackup.s3.bucket"
                        touched={touched}
                        errors={errors}
                        helperText="example, acs.backups"
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.bucket"
                            value={values.externalBackup.s3.bucket}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Object prefix"
                        fieldId="externalBackup.s3.objectPrefix"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.objectPrefix"
                            value={values.externalBackup.s3.objectPrefix}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        fieldId="externalBackup.s3.endpoint"
                        helperText="example, s3.us-west-2.amazonaws.com"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.endpoint"
                            value={values.externalBackup.s3.endpoint}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Region"
                        fieldId="externalBackup.s3.region"
                        helperText="example, us-west-2"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3.region"
                            value={values.externalBackup.s3.region}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label=""
                        fieldId="externalBackup.s3.useIam"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Use container IAM role"
                            id="externalBackup.s3.useIam"
                            isChecked={values.externalBackup.s3.useIam}
                            onChange={updateKeysOnChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label=""
                        fieldId="externalBackup.s3.usePathstyle"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Force path style URLs for S3 objects"
                            id="externalBackup.s3.usePathstyle"
                            isChecked={values.externalBackup.s3.usePathstyle}
                            onChange={updateKeysOnChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
                        <FormLabelGroup
                            label=""
                            fieldId="updatePassword"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            errors={errors}
                        >
                            <Checkbox
                                label="Update access key ID and secret access key"
                                id="updatePassword"
                                isChecked={values.updatePassword}
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Access key ID"
                        fieldId="externalBackup.s3.accessKeyId"
                        isRequired={values.updatePassword && !values.externalBackup.s3.useIam}
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword && !values.externalBackup.s3.useIam}
                            type="password"
                            id="externalBackup.s3.accessKeyId"
                            value={values.externalBackup.s3.accessKeyId}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={
                                !isEditable ||
                                !values.updatePassword ||
                                values.externalBackup.s3.useIam
                            }
                            placeholder={
                                values.updatePassword || values.externalBackup.s3.useIam
                                    ? ''
                                    : 'Currently-stored access key ID will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Secret access key"
                        fieldId="externalBackup.s3.secretAccessKey"
                        isRequired={values.updatePassword && !values.externalBackup.s3.useIam}
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword && !values.externalBackup.s3.useIam}
                            type="password"
                            id="externalBackup.s3.secretAccessKey"
                            value={values.externalBackup.s3.secretAccessKey}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={
                                !isEditable ||
                                !values.updatePassword ||
                                values.externalBackup.s3.useIam
                            }
                            placeholder={
                                values.updatePassword || values.externalBackup.s3.useIam
                                    ? ''
                                    : 'Currently-stored secret access key will be used.'
                            }
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={onTest}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!isValid}
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

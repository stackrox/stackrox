/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import {
    Checkbox,
    Form,
    FormSelect,
    PageSection,
    Popover,
    TextInput,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { HelpIcon } from '@patternfly/react-icons';
import * as yup from 'yup';

import { BackupIntegrationBase } from 'services/BackupIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import SelectSingle from 'Components/SelectSingle';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import ScheduleIntervalOptions from '../FormSchedule/ScheduleIntervalOptions';
import ScheduleWeeklyOptions from '../FormSchedule/ScheduleWeeklyOptions';
import ScheduleDailyOptions from '../FormSchedule/ScheduleDailyOptions';

const urlStyles = [
    {
        label: 'Path',
        value: 'S3_URL_STYLE_PATH',
    },
    {
        label: 'Virtual hosted',
        value: 'S3_URL_STYLE_VIRTUAL_HOSTED',
    },
];

export type S3CompatibleIntegration = {
    s3compatible: {
        bucket: string;
        objectPrefix: string;
        endpoint: string;
        region: string;
        accessKeyId: string;
        secretAccessKey: string;
        urlStyle: 'S3_URL_STYLE_PATH' | 'S3_URL_STYLE_VIRTUAL_HOSTED';
    };
    type: 's3compatible';
} & BackupIntegrationBase;

export type S3CompatibleIntegrationFormValues = {
    externalBackup: S3CompatibleIntegration;
    updatePassword: boolean;
};

function requireCredentials(value, context: yup.TestContext) {
    const requirePasswordField =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        context?.from[2]?.value?.updatePassword || false;

    if (!requirePasswordField) {
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
        s3compatible: yup.object().shape({
            bucket: yup.string().trim().required('Bucket is required'),
            objectPrefix: yup.string(),
            endpoint: yup.string(),
            region: yup.string().trim().required('Region is required'),
            urlStyle: yup.string().trim().required('URL style is required'),
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
        type: yup.string().matches(/s3compatible/),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: S3CompatibleIntegrationFormValues = {
    externalBackup: {
        id: '',
        name: '',
        backupsToKeep: 1,
        schedule: {
            intervalType: 'DAILY',
            hour: 0,
            minute: 0,
        },
        s3compatible: {
            bucket: '',
            objectPrefix: '',
            endpoint: '',
            region: '',
            accessKeyId: '',
            secretAccessKey: '',
            urlStyle: 'S3_URL_STYLE_PATH',
        },
        type: 's3compatible',
    },
    updatePassword: true,
};

function urlStyleIcon(): ReactElement {
    return (
        <Popover
            bodyContent={
                <div>
                    <a
                        href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html"
                        target="_blank"
                        rel="noreferrer"
                    >
                        The URL style defines the bucket URL addressing. Virtual-hosted-style
                        buckets are addressed as https://&#60;bucket&#62;.&#60;endpoint&#62; while
                        path-style buckets are addressed as
                        https://&#60;endpoint&#62;/&#60;bucket&#62;. See the AWS documentation about
                        virtual hosting for more information.
                    </a>
                </div>
            }
            headerContent={'Virtual hosting of buckets'}
        >
            <button
                type="button"
                aria-label="More info for input"
                onClick={(e) => e.preventDefault()}
                aria-describedby="simple-form-name-01"
                className="pf-v5-c-form__group-label-help"
            >
                <HelpIcon />
            </button>
        </Popover>
    );
}

function S3CompatibleIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<S3CompatibleIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };

    if (initialValues) {
        formInitialValues.externalBackup = {
            ...formInitialValues.externalBackup,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.externalBackup.s3compatible.accessKeyId = '';
        formInitialValues.externalBackup.s3compatible.secretAccessKey = '';

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
    } = useIntegrationForm<S3CompatibleIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('externalBackup.s3compatible.accessKeyId', '');
        setFieldValue('externalBackup.s3compatible.secretAccessKey', '');
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
                            onChange={(event, value) => onChange(value, event)}
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
                            onChange={(event, value) => onChange(value, event)}
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
                            onChange={(event, value) => onChange(value, event)}
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
                                onChange={(event, value) => onChange(value, event)}
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            <ScheduleDailyOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Bucket"
                        fieldId="externalBackup.s3compatible.bucket"
                        touched={touched}
                        errors={errors}
                        helperText="example, acs.backups"
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3compatible.bucket"
                            value={values.externalBackup.s3compatible.bucket}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Object prefix"
                        fieldId="externalBackup.s3compatible.objectPrefix"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3compatible.objectPrefix"
                            value={values.externalBackup.s3compatible.objectPrefix}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Endpoint"
                        fieldId="externalBackup.s3compatible.endpoint"
                        helperText="example, play.min.io"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3compatible.endpoint"
                            value={values.externalBackup.s3compatible.endpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Region"
                        fieldId="externalBackup.s3compatible.region"
                        helperText="example, us-west-2"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.s3compatible.region"
                            value={values.externalBackup.s3compatible.region}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="URL style"
                        labelIcon={urlStyleIcon()}
                        fieldId="externalBackup.s3compatible.urlStyle"
                        errors={errors}
                    >
                        <SelectSingle
                            id="externalBackup.s3compatible.urlStyle"
                            value={values.externalBackup.s3compatible.urlStyle}
                            handleSelect={setFieldValue}
                            direction="up"
                        >
                            {urlStyles.map(({ value, label }) => (
                                <SelectOption key={value} value={value}>
                                    {label}
                                </SelectOption>
                            ))}
                        </SelectSingle>
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
                                onChange={(event, value) => onUpdateCredentialsChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Access key ID"
                        fieldId="externalBackup.s3compatible.accessKeyId"
                        isRequired={values.updatePassword}
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword}
                            type="password"
                            id="externalBackup.s3compatible.accessKeyId"
                            value={values.externalBackup.s3compatible.accessKeyId}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updatePassword}
                            placeholder={
                                values.updatePassword
                                    ? ''
                                    : 'Currently-stored access key ID will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Secret access key"
                        fieldId="externalBackup.s3compatible.secretAccessKey"
                        isRequired={values.updatePassword}
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword}
                            type="password"
                            id="externalBackup.s3compatible.secretAccessKey"
                            value={values.externalBackup.s3compatible.secretAccessKey}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updatePassword}
                            placeholder={
                                values.updatePassword
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

export default S3CompatibleIntegrationForm;

/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import {
    Checkbox,
    Form,
    FormSelect,
    PageSection,
    TextInput,
    TextArea,
} from '@patternfly/react-core';
import * as yup from 'yup';

import { BackupIntegrationBase } from 'services/BackupIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import ScheduleIntervalOptions from '../FormSchedule/ScheduleIntervalOptions';
import ScheduleWeeklyOptions from '../FormSchedule/ScheduleWeeklyOptions';
import ScheduleDailyOptions from '../FormSchedule/ScheduleDailyOptions';

import { getGoogleCredentialsPlaceholder } from '../../utils/integrationUtils';

export type GcsIntegration = {
    gcs: {
        bucket: string;
        objectPrefix: string;
        useWorkloadId: boolean;
        serviceAccount: string;
    };
    type: 'gcs';
} & BackupIntegrationBase;

export type GcsIntegrationFormValues = {
    externalBackup: GcsIntegration;
    updatePassword: boolean;
};

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
        gcs: yup.object().shape({
            bucket: yup.string().trim().required('Bucket is required'),
            objectPrefix: yup.string().trim(),
            useWorkloadId: yup.bool(),
            serviceAccount: yup
                .string()
                .trim()
                .test(
                    'serviceAccount-test',
                    'Valid JSON is required for service account key',
                    (value, context: yup.TestContext) => {
                        const requirePasswordField =
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            context?.from[2]?.value?.updatePassword || false;
                        const useWorkloadId = context?.parent?.useWorkloadId;

                        if (!requirePasswordField || useWorkloadId) {
                            return true;
                        }
                        try {
                            JSON.parse(value as string);
                        } catch (e) {
                            return false;
                        }
                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
        }),
        type: yup.string().matches(/gcs/),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: GcsIntegrationFormValues = {
    externalBackup: {
        id: '',
        name: '',
        backupsToKeep: 1,
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
    },
    updatePassword: true,
};

function GcsIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GcsIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.externalBackup = {
            ...formInitialValues.externalBackup,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.externalBackup.gcs.serviceAccount = '';

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
    } = useIntegrationForm<GcsIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function updateServiceAccountOnChange(value, event) {
        void setFieldValue(event.target.id, value);
        if (value === true) {
            void setFieldValue('externalBackup.gcs.serviceAccount', '');
        }
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('externalBackup.gcs.serviceAccount', '');
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
                            name="externalBackup.backupsToKeep"
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
                        fieldId="externalBackup.gcs.bucket"
                        touched={touched}
                        errors={errors}
                        helperText="example, stackrox.backups"
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.gcs.bucket"
                            name="externalBackup.gcs.bucket"
                            value={values.externalBackup.gcs.bucket}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Object prefix"
                        fieldId="externalBackup.gcs.objectPrefix"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="externalBackup.gcs.objectPrefix"
                            name="externalBackup.gcs.objectPrefix"
                            value={values.externalBackup.gcs.objectPrefix}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label=""
                        fieldId="externalBackup.gcs.useWorkloadId"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Use workload identity"
                            id="externalBackup.gcs.useWorkloadId"
                            isChecked={values.externalBackup.gcs.useWorkloadId}
                            onChange={updateServiceAccountOnChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
                        <FormLabelGroup
                            label=""
                            fieldId="updatePassword"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            touched={touched}
                            errors={errors}
                        >
                            <Checkbox
                                label="Update stored credentials"
                                id="updatePassword"
                                isChecked={
                                    !values.externalBackup.gcs.useWorkloadId &&
                                    values.updatePassword
                                }
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || values.externalBackup.gcs.useWorkloadId}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Service account key (JSON)"
                        isRequired={
                            values.updatePassword && !values.externalBackup.gcs.useWorkloadId
                        }
                        fieldId="externalBackup.gcs.serviceAccount"
                        touched={touched}
                        errors={errors}
                    >
                        <TextArea
                            className="json-input"
                            isRequired={
                                values.updatePassword && !values.externalBackup.gcs.useWorkloadId
                            }
                            type="text"
                            id="externalBackup.gcs.serviceAccount"
                            name="externalBackup.gcs.serviceAccount"
                            value={values.externalBackup.gcs.serviceAccount}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={
                                !isEditable ||
                                !values.updatePassword ||
                                values.externalBackup.gcs.useWorkloadId
                            }
                            placeholder={getGoogleCredentialsPlaceholder(
                                values.externalBackup.gcs.useWorkloadId,
                                values.updatePassword
                            )}
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

export default GcsIntegrationForm;

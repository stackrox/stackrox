import type { ReactElement } from 'react';
import {
    Checkbox,
    Content,
    Form,
    List,
    ListItem,
    PageSection,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

import type { BackupIntegrationBase } from 'services/BackupIntegrationsService';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import SelectSingle from 'Components/SelectSingle';

import usePageState from '../../../hooks/usePageState';
import useIntegrationForm from '../../useIntegrationForm';
import type { IntegrationFormProps } from '../../integrationFormTypes';

import IntegrationFormActions from '../../IntegrationFormActions';
import FormLabelGroup from '../../FormLabelGroup';

import IntegrationHelpIcon from '../Components/IntegrationHelpIcon';

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

const intervalOptions = [
    { label: 'Daily', value: 'DAILY' },
    { label: 'Weekly', value: 'WEEKLY' },
];

const weeklyOptions = [
    { label: 'Sunday', value: '0' },
    { label: 'Monday', value: '1' },
    { label: 'Tuesday', value: '2' },
    { label: 'Wednesday', value: '3' },
    { label: 'Thursday', value: '4' },
    { label: 'Friday', value: '5' },
    { label: 'Saturday', value: '6' },
];

const dailyOptions = Array.from({ length: 24 }, (_, i) => ({
    label: `${String(i).padStart(2, '0')}:00 UTC`,
    value: String(i),
}));

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

function S3CompatibleIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<S3CompatibleIntegration>): ReactElement {
    const formInitialValues = structuredClone(defaultValues);
    if (initialValues) {
        merge(formInitialValues.externalBackup, initialValues);

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

    function onScheduleIntervalChange(fieldId, value) {
        // Initialize weekly day when switching to WEEKLY
        if (value === 'WEEKLY' && !values.externalBackup.schedule.weekly) {
            setFieldValue('externalBackup.schedule', {
                ...values.externalBackup.schedule,
                intervalType: value,
                weekly: { day: 0 },
            });
        } else {
            setFieldValue(fieldId, value);
        }
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('externalBackup.s3compatible.accessKeyId', '');
        setFieldValue('externalBackup.s3compatible.secretAccessKey', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection hasBodyWrapper={false} isFilled hasOverflowScroll>
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
                        <SelectSingle
                            id="externalBackup.schedule.intervalType"
                            value={values.externalBackup.schedule.intervalType}
                            handleSelect={onScheduleIntervalChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            {intervalOptions.map(({ value, label }) => (
                                <SelectOption key={value} value={value}>
                                    {label}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FormLabelGroup>
                    {values.externalBackup.schedule.intervalType === 'WEEKLY' && (
                        <FormLabelGroup
                            isRequired
                            label="Schedule day of week"
                            fieldId="externalBackup.schedule.weekly.day"
                            touched={touched}
                            errors={errors}
                        >
                            <SelectSingle
                                id="externalBackup.schedule.weekly.day"
                                value={String(values.externalBackup.schedule?.weekly?.day ?? 0)}
                                handleSelect={setFieldValue}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            >
                                {weeklyOptions.map(({ value, label }) => (
                                    <SelectOption key={value} value={value}>
                                        {label}
                                    </SelectOption>
                                ))}
                            </SelectSingle>
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired
                        label="Schedule time of day"
                        fieldId="externalBackup.schedule.hour"
                        touched={touched}
                        errors={errors}
                    >
                        <SelectSingle
                            id="externalBackup.schedule.hour"
                            value={String(values.externalBackup.schedule.hour)}
                            handleSelect={setFieldValue}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            {dailyOptions.map(({ value, label }) => (
                                <SelectOption key={value} value={value}>
                                    {label}
                                </SelectOption>
                            ))}
                        </SelectSingle>
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
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Object prefix"
                                helpText={
                                    <div>
                                        Creates a new folder &#60;prefix&#62; under which backups
                                        files are placed.
                                    </div>
                                }
                                ariaLabel="Help for object prefix"
                            />
                        }
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
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Endpoint"
                                helpText={
                                    <div>
                                        The endpoint under which the S3 compatible service is
                                        reached. Defaults to https if the scheme is left out. Note
                                        that when using AWS S3, it is recommended to create an{' '}
                                        <em>Amazon S3</em> integration instead.
                                    </div>
                                }
                                ariaLabel="Help for endpoint"
                            />
                        }
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
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Region"
                                helpText={
                                    <div>
                                        Consult the service provider&apos;s S3 compatibility
                                        instructions for the correct region.
                                    </div>
                                }
                                ariaLabel="Help for region"
                            />
                        }
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
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Virtual hosting of buckets"
                                helpText={
                                    <>
                                        <Content component="p">
                                            Defines the bucket URL addressing:
                                        </Content>
                                        <List className="pf-v6-u-py-sm">
                                            <ListItem>
                                                Virtual-hosted-style buckets are addressed as
                                                https://&#60;bucket&#62;.&#60;endpoint&#62
                                            </ListItem>
                                            <ListItem>
                                                Path-style buckets are addressed as
                                                https://&#60;endpoint&#62;/&#60;bucket&#62;
                                            </ListItem>
                                        </List>
                                        <Content component="p">
                                            For more information, see{' '}
                                            <ExternalLink>
                                                <a
                                                    href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html"
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                >
                                                    AWS documentation about virtual hosting
                                                </a>
                                            </ExternalLink>
                                        </Content>
                                    </>
                                }
                                ariaLabel="Help for URL style"
                            />
                        }
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

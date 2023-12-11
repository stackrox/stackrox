import React, { ReactElement } from 'react';
import { Checkbox, Form, PageSection, TextInput, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import useFeatureFlags from '../../../../hooks/useFeatureFlags';

export type GoogleCloudSccIntegration = {
    cscc: {
        serviceAccount: string;
        sourceId: string;
        wifEnabled: boolean;
    };
    type: 'cscc';
} & NotifierIntegrationBase;

export type GoogleCloudSccIntegrationFormValues = {
    notifier: GoogleCloudSccIntegration;
    updatePassword: boolean;
};

const sourceIdRegex = /^organizations\/[0-9]+\/sources\/[0-9]+$/;

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('An integration name is required'),
        cscc: yup.object().shape({
            wifEnabled: yup.bool(),
            serviceAccount: yup
                .string()
                .trim()
                .when('wifEnabled', {
                    is: false,
                    then: (serviceAccountSchema) =>
                        serviceAccountSchema
                            .required('A service account is required')
                            .test(
                                'isValidJson',
                                'Service account must be valid JSON',
                                (value, context: yup.TestContext) => {
                                    const isRequired =
                                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                        // @ts-ignore
                                        context?.from[2]?.value?.updatePassword || false;
                                    if (!isRequired) {
                                        return true;
                                    }
                                    if (!value) {
                                        return false;
                                    }
                                    try {
                                        JSON.parse(value);
                                    } catch (e) {
                                        return false;
                                    }
                                    return true;
                                }
                            ),
                }),
            sourceId: yup
                .string()
                .trim()
                .required('A source ID is required')
                .matches(
                    sourceIdRegex,
                    'SCC source ID must match the format: organizations/[0-9]+/sources/[0-9]+'
                ),
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: GoogleCloudSccIntegrationFormValues = {
    notifier: {
        id: '',
        name: '',
        cscc: {
            serviceAccount: '',
            sourceId: '',
            wifEnabled: false,
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'cscc',
    },
    updatePassword: true,
};

function GoogleCloudSccIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GoogleCloudSccIntegration>): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showWIFCheckbox = isFeatureFlagEnabled('ROX_CLOUD_CREDENTIALS');
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.notifier = {
            ...formInitialValues.notifier,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.cscc.serviceAccount = '';

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
    } = useIntegrationForm<GoogleCloudSccIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('notifier.cscc.serviceAccount', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="notifier.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.name"
                            value={values.notifier.name}
                            placeholder="(example, Cloud SCC Integration)"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Cloud SCC Source ID"
                        isRequired
                        fieldId="notifier.cscc.sourceId"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.cscc.sourceId"
                            value={values.notifier.cscc.sourceId}
                            placeholder="example, organizations/123/sources/456"
                            onChange={onChange}
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
                                label="Update token"
                                id="updatePassword"
                                isChecked={values.updatePassword}
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    {showWIFCheckbox && (
                        <FormLabelGroup
                            fieldId="notifier.cscc.wifEnabled"
                            touched={touched}
                            errors={errors}
                        >
                            <Checkbox
                                label="Enable WIF"
                                id="notifier.cscc.wifEnabled"
                                aria-label="enable wif"
                                isChecked={values.notifier.cscc.wifEnabled}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    {!values.notifier.cscc.wifEnabled && (
                        <FormLabelGroup
                            label="Service Account Key (JSON)"
                            isRequired={values.updatePassword}
                            fieldId="notifier.cscc.serviceAccount"
                            touched={touched}
                            errors={errors}
                        >
                            <TextArea
                                className="json-input"
                                isRequired={values.updatePassword}
                                type="text"
                                id="notifier.cscc.serviceAccount"
                                value={values.notifier.cscc.serviceAccount}
                                placeholder={
                                    values.updatePassword
                                        ? 'example,\n{\n  "type": "service_account",\n  "project_id": "123456"\n  ...\n}'
                                        : 'Currently-stored credentials will be used.'
                                }
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || !values.updatePassword}
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

export default GoogleCloudSccIntegrationForm;

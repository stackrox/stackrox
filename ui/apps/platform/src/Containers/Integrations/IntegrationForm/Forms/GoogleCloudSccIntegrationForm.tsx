import React, { ReactElement } from 'react';
import { Checkbox, Form, PageSection, TextInput, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

import { getGoogleCredentialsPlaceholder } from '../../utils/integrationUtils';

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
                .test(
                    'serviceAccount-test',
                    'Valid JSON is required for service account key',
                    (value, context: yup.TestContext) => {
                        const requirePasswordField =
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            context?.from[2]?.value?.updatePassword || false;
                        const useWorkloadId = context?.parent?.wifEnabled;

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
    const isCloudCredentialsEnabled = isFeatureFlagEnabled('ROX_CLOUD_CREDENTIALS');
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
                    {isCloudCredentialsEnabled && (
                        <FormLabelGroup
                            fieldId="notifier.cscc.wifEnabled"
                            touched={touched}
                            errors={errors}
                        >
                            <Checkbox
                                label="Use workload identity"
                                id="notifier.cscc.wifEnabled"
                                isChecked={values.notifier.cscc.wifEnabled}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
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
                                    !(
                                        isCloudCredentialsEnabled && values.notifier.cscc.wifEnabled
                                    ) && values.updatePassword
                                }
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={
                                    !isEditable ||
                                    (isCloudCredentialsEnabled && values.notifier.cscc.wifEnabled)
                                }
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Service account key (JSON)"
                        isRequired={
                            values.updatePassword &&
                            isCloudCredentialsEnabled &&
                            !values.notifier.cscc.wifEnabled
                        }
                        fieldId="notifier.cscc.serviceAccount"
                        touched={touched}
                        errors={errors}
                    >
                        <TextArea
                            className="json-input"
                            isRequired={
                                values.updatePassword &&
                                !(isCloudCredentialsEnabled && values.notifier.cscc.wifEnabled)
                            }
                            type="text"
                            id="notifier.cscc.serviceAccount"
                            name="notifier.cscc.serviceAccount"
                            value={values.notifier.cscc.serviceAccount}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={
                                !isEditable ||
                                !values.updatePassword ||
                                (isCloudCredentialsEnabled && values.notifier.cscc.wifEnabled)
                            }
                            placeholder={getGoogleCredentialsPlaceholder(
                                isCloudCredentialsEnabled && values.notifier.cscc.wifEnabled,
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

export default GoogleCloudSccIntegrationForm;

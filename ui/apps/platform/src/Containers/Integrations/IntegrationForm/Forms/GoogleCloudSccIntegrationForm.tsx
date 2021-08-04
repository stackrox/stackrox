import React, { ReactElement } from 'react';
import { Form, PageSection, TextInput, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';

export type GoogleCloudSccIntegration = {
    id?: string;
    name: string;
    cscc: {
        serviceAccount: string;
        sourceId: string;
    };
    uiEndpoint: string;
    type: 'cscc';
    enabled: boolean;
};

const sourceIdRegex = /^organizations\/[0-9]+\/sources\/[0-9]+$/;

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Required'),
    cscc: yup.object().shape({
        serviceAccount: yup
            .string()
            .trim()
            .required('A service account is required')
            .test('isValidJson', 'Service account must be valid JSON', (value) => {
                if (!value) {
                    return false;
                }
                try {
                    JSON.parse(value);
                } catch (e) {
                    return false;
                }
                return true;
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
});

export const defaultValues: GoogleCloudSccIntegration = {
    name: '',
    cscc: {
        serviceAccount: '',
        sourceId: '',
    },
    uiEndpoint: window.location.origin,
    type: 'cscc',
    enabled: true,
};

function GoogleCloudSccIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GoogleCloudSccIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? { ...defaultValues, ...initialValues }
        : defaultValues;
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
    } = useIntegrationForm<GoogleCloudSccIntegration, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            placeholder="(example, Cloud SCC Integration)"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Cloud SCC Source ID"
                        isRequired
                        fieldId="cscc.sourceId"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="cscc.sourceId"
                            value={values.cscc.sourceId}
                            placeholder="example, organizations/123/sources/456"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Service Account Key (JSON)"
                        isRequired
                        fieldId="cscc.serviceAccount"
                        touched={touched}
                        errors={errors}
                    >
                        <TextArea
                            className="json-input"
                            isRequired
                            type="text"
                            id="cscc.serviceAccount"
                            value={values.cscc.serviceAccount}
                            placeholder={
                                'example,\n{\n  "type": "service_account",\n  "project_id": "123456"\n  ...\n}'
                            }
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
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
                        isValid={isValid}
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

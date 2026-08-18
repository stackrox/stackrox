import type { ReactElement } from 'react';
import * as yup from 'yup';
import { Form, PageSection, TextInput } from '@patternfly/react-core';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import type { AiIntegration } from 'services/AiIntegrationsService';
import merge from 'lodash/merge';

import FormLabelGroup from '../../FormLabelGroup';
import IntegrationFormActions from '../../IntegrationFormActions';
import useIntegrationForm from '../../useIntegrationForm';
import type { IntegrationFormProps } from '../../integrationFormTypes';

export const validationSchema = yup.object().shape({
    integration: yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        type: yup.string().matches(/AI_INTEGRATION_TYPE_OLS/),
        serviceUrl: yup.string().trim(),
    }),
});

export type AiIntegrationFormValues = {
    integration: AiIntegration;
};

export const defaultValues: AiIntegrationFormValues = {
    integration: {
        id: '',
        name: '',
        type: 'AI_INTEGRATION_TYPE_OLS',
        serviceUrl: '',
    },
};

function LightspeedIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<AiIntegration>): ReactElement {
    const formInitialValues = structuredClone(defaultValues);
    if (initialValues) {
        merge(formInitialValues.integration, initialValues);
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
    } = useIntegrationForm<AiIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        isRequired
                        label="Integration name"
                        fieldId="integration.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="integration.name"
                            value={values.integration.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Service URL"
                        fieldId="integration.serviceUrl"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="integration.serviceUrl"
                            value={values.integration.serviceUrl}
                            onChange={(event, value) => onChange(value, event)}
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

export default LightspeedIntegrationForm;

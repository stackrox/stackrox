import type { ReactElement } from 'react';
import * as yup from 'yup';
import { Form, PageSection, TextInput } from '@patternfly/react-core';
import merge from 'lodash/merge';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import type { AiIntegration } from 'services/AiIntegrationsService';

import FormLabelGroup from '../../FormLabelGroup';
import IntegrationFormActions from '../../IntegrationFormActions';
import useIntegrationForm from '../../useIntegrationForm';
import type { IntegrationFormProps } from '../../integrationFormTypes';

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Integration name is required'),
    type: yup.string().oneOf(['AI_INTEGRATION_TYPE_OLS']).required(),
    serviceUrl: yup.string().trim(),
});

export type AiIntegrationFormValues = AiIntegration;

export const defaultValues: AiIntegrationFormValues = {
    id: '',
    name: '',
    type: 'AI_INTEGRATION_TYPE_OLS',
    serviceUrl: '',
};

function LightspeedIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<AiIntegration>): ReactElement {
    const formInitialValues = structuredClone(defaultValues);
    if (initialValues) {
        merge(formInitialValues, initialValues);
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
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Service URL"
                        fieldId="serviceUrl"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="serviceUrl"
                            value={values.serviceUrl}
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

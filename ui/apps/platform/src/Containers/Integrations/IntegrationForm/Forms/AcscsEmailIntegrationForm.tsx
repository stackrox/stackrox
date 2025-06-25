import React, { ReactElement } from 'react';
import { Form, PageSection, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type ACSCSEmailIntegration = {
    type: 'acscsEmail';
} & NotifierIntegrationBase;

export type ACSCSEmailIntegrationFormValues = ACSCSEmailIntegration;

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Email integration name is required'),
    labelDefault: yup
        .string()
        .trim()
        .required('A default recipient email address is required')
        .email('Must be a valid default recipient email address'),
    labelKey: yup.string(),
});

export const defaultValues: ACSCSEmailIntegrationFormValues = {
    id: '',
    name: '',
    type: 'acscsEmail',
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
};

function AcscsEmailIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ACSCSEmailIntegration>): ReactElement {
    const formInitialValues: ACSCSEmailIntegrationFormValues = merge(
        {},
        defaultValues,
        initialValues
    );
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
    } = useIntegrationForm<ACSCSEmailIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
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
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            placeholder="(example, Email Integration)"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Default recipient"
                        fieldId="labelDefault"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="labelDefault"
                            value={values.labelDefault}
                            placeholder="example, security-alerts-recipients@example.com"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Annotation key for recipient"
                        labelIcon={<AnnotationKeyLabelIcon />}
                        fieldId="labelKey"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="labelKey"
                            value={values.labelKey}
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

export default AcscsEmailIntegrationForm;

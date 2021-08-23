import React, { ReactElement } from 'react';
import { Form, PageSection, Switch, TextInput } from '@patternfly/react-core';
import merge from 'lodash/merge';
import * as yup from 'yup';

import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';

export type SumoLogicIntegration = {
    id: string;
    name: string;
    type: 'sumologic';
    uiEndpoint: string;
    labelKey: string;
    labelDefault: string;
    sumologic: {
        httpSourceAddress: string;
        skipTLSVerify: boolean;
    };
};

const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Name is required'),
    sumologic: yup.object().shape({
        httpSourceAddress: yup
            .string()
            .trim()
            .required('HTTP Collector Source Address is required'),
        skipTLSVerify: yup.bool(),
    }),
});

const defaultValues: SumoLogicIntegration = {
    id: '',
    name: '',
    type: 'sumologic',
    uiEndpoint: window.location.origin,
    labelKey: '',
    labelDefault: '',
    sumologic: {
        httpSourceAddress: '',
        skipTLSVerify: false,
    },
};

function SumoLogicIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<SumoLogicIntegration>): ReactElement {
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
    } = useIntegrationForm<SumoLogicIntegration, typeof validationSchema>({
        initialValues: merge({}, defaultValues, initialValues), // in case properties are missing from initialValues
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
                        isRequired
                        label="Name"
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            name="name"
                            value={values.name}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="HTTP Collector Source Address"
                        fieldId="sumologic.httpSourceAddress"
                        touched={touched}
                        errors={errors}
                        helperText="For example, https://endpoint.sumologic.com/receiver/v1/http/<token>"
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="sumologic.httpSourceAddress"
                            name="sumologic.httpSourceAddress"
                            value={values.sumologic.httpSourceAddress}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Disable TLS Certificate Validation (Insecure)"
                        fieldId="sumologic.skipTLSVerify"
                        errors={errors}
                    >
                        <Switch
                            id="sumologic.skipTLSVerify"
                            name="sumologic.skipTLSVerify"
                            aria-label="disable tls certificate validation"
                            isChecked={values.sumologic.skipTLSVerify}
                            onChange={onChange}
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

export default SumoLogicIntegrationForm;

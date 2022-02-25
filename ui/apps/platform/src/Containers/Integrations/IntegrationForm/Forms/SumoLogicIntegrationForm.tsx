import React, { ReactElement } from 'react';
import { Checkbox, Form, PageSection, TextInput } from '@patternfly/react-core';
import merge from 'lodash/merge';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type SumoLogicIntegration = {
    sumologic: {
        httpSourceAddress: string;
        skipTLSVerify: boolean;
    };
    type: 'sumologic';
} & NotifierIntegrationBase;

const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Integration name is required'),
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
    sumologic: {
        httpSourceAddress: '',
        skipTLSVerify: false,
    },
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
    type: 'sumologic',
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
    } = useIntegrationForm<SumoLogicIntegration>({
        initialValues: merge({}, defaultValues, initialValues), // in case properties are missing from initialValues
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
                    <FormLabelGroup label="" fieldId="sumologic.skipTLSVerify" errors={errors}>
                        <Checkbox
                            label="Disable TLS certificate validation (insecure)"
                            id="sumologic.skipTLSVerify"
                            isChecked={values.sumologic.skipTLSVerify}
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

export default SumoLogicIntegrationForm;

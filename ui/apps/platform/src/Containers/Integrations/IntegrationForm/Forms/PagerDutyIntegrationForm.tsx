import React, { ReactElement, useState } from 'react';
import { Checkbox, Form, PageSection, TextInput } from '@patternfly/react-core';
import merge from 'lodash/merge';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import usePageState from '../../hooks/usePageState';
import { clearStoredCredentials } from '../../utils/integrationUtils';

import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type PagerDutyIntegration = {
    type: 'pagerduty';
    pagerduty: {
        apiKey: string;
    };
} & NotifierIntegrationBase;

export const defaultValues: PagerDutyIntegration = {
    id: '',
    name: '',
    type: 'pagerduty',
    uiEndpoint: window.location.origin,
    labelDefault: '',
    labelKey: '',
    pagerduty: {
        apiKey: '',
    },
};

const storedCredentialKeyPaths = ['pagerduty.apiKey'];

function getSchema(isStoredCredentialRequired: boolean) {
    return yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        pagerduty: yup.object().shape({
            apiKey: isStoredCredentialRequired
                ? yup.string().required('PagerDuty integration key is required')
                : yup.string(),
        }),
    });
}

const validationSchemaStoredCredentialRequired = getSchema(true);
const validationSchemaStoredCredentialNotRequired = getSchema(false);

function PagerDutyIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<PagerDutyIntegration>): ReactElement {
    const { isCreating } = usePageState();

    const [isUpdatingStoredCredential, setIsUpdatingCredential] = useState(false);
    const isStoredCredentialInputEnabled = isCreating || isUpdatingStoredCredential;

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
    } = useIntegrationForm<PagerDutyIntegration>({
        initialValues: initialValues
            ? clearStoredCredentials(
                  merge({}, defaultValues, initialValues),
                  storedCredentialKeyPaths
              )
            : defaultValues,
        validationSchema: () =>
            isStoredCredentialInputEnabled
                ? validationSchemaStoredCredentialRequired
                : validationSchemaStoredCredentialNotRequired,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onChangeUpdateStoredCredential(value) {
        setIsUpdatingCredential(value);
        if (!value) {
            // Clear credential text because checkbox has been cleared.
            storedCredentialKeyPaths.forEach((keyPath) => {
                setFieldValue(keyPath, '');
            });
        }
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
                    {!isCreating && isEditable && (
                        <FormLabelGroup label="" fieldId="updateStoredCredential" errors={errors}>
                            <Checkbox
                                label="Update PagerDuty Integration Key"
                                id="updateStoredCredential"
                                name="updateStoredCredential"
                                isChecked={isUpdatingStoredCredential}
                                onChange={onChangeUpdateStoredCredential}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired
                        label="PagerDuty integration key"
                        fieldId="pagerduty.apiKey"
                        touched={touched}
                        errors={isStoredCredentialInputEnabled ? errors : {}}
                    >
                        <TextInput
                            isRequired
                            type="password"
                            id="pagerduty.apiKey"
                            name="pagerduty.apiKey"
                            value={values.pagerduty.apiKey}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isStoredCredentialInputEnabled}
                            placeholder={
                                isStoredCredentialInputEnabled
                                    ? ''
                                    : 'This integration has stored credentials'
                            }
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={() =>
                            onSave(isCreating ? {} : { updatePassword: isUpdatingStoredCredential })
                        }
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={() =>
                            onTest(isCreating ? {} : { updatePassword: isUpdatingStoredCredential })
                        }
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

export default PagerDutyIntegrationForm;

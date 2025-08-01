import React, { ReactElement } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import {
    ActionGroup,
    Alert,
    Button,
    Divider,
    Flex,
    Form,
    PageSection,
    TextInput,
} from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useAnalytics, { DOWNLOAD_CLUSTER_REGISTRATION_SECRET } from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import { generateClusterRegistrationSecret } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterRegistrationSecretsHeader from './ClusterRegistrationSecretsHeader';

import { downloadClusterRegistrationSecret } from './ClusterRegistrationSecretForm.utils';

export type ClusterRegistrationSecretFormValues = {
    name: string;
};

export const initialValues: ClusterRegistrationSecretFormValues = {
    name: '',
};

// https://github.com/stackrox/stackrox/blob/master/central/clusterinit/backend/validation.go#L11
const nameValidatorRegExp = /^[a-zA-Z0-9._-]+$/;

const validationSchema: yup.ObjectSchema<ClusterRegistrationSecretFormValues> = yup.object().shape({
    name: yup
        .string()
        .trim()
        .matches(
            nameValidatorRegExp,
            'Name can have only the following characters: letters, digits, period, underscore, hyphen (but no spaces)'
        )
        .required('Cluster registration secret name is required'),
});

function ClusterRegistrationSecretForm(): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();

    const { mutate, error } = useRestMutation(
        (name: string) => generateClusterRegistrationSecret({ name }),
        {
            onSuccess: (response) => {
                downloadClusterRegistrationSecret(values.name, response);
                goBack();
            },
            onSettled: () => setSubmitting(false),
        }
    );
    const {
        errors,
        handleBlur,
        isSubmitting,
        isValid,
        setFieldValue,
        submitForm,
        touched,
        values,
        setSubmitting,
    } = useFormik({
        initialValues,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(true);
            mutate(values.name);
        },
        validateOnMount: true, // disable Next when Name is empty
        validationSchema,
    });

    function goBack() {
        navigate(-1); // to ClusterRegistrationSecrets Table
    }

    function onChangeTextInput(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <ClusterRegistrationSecretsHeader title="Create cluster registration secret" />
            <Divider component="div" />
            <PageSection variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Form>
                        <FormLabelGroup
                            fieldId="name"
                            label="Name"
                            isRequired
                            errors={errors}
                            touched={touched}
                        >
                            <TextInput
                                type="text"
                                id="name"
                                name="name"
                                isRequired
                                value={values.name}
                                onBlur={handleBlur}
                                onChange={(event, value) => onChangeTextInput(value, event)}
                            />
                        </FormLabelGroup>
                    </Form>
                    <Alert variant="info" isInline title="Download YAML file" component="p">
                        <p>
                            You can download the YAML file only once, when you create a cluster
                            registration secret.
                        </p>
                        <p>Store the YAML file securely because it contains secrets.</p>
                    </Alert>
                    {error !== undefined && (
                        <Alert
                            variant="danger"
                            isInline
                            title="Unable to create or download cluster registration secret"
                            component="p"
                        >
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    )}
                    <ActionGroup>
                        <Button
                            variant="primary"
                            isDisabled={isSubmitting || !isValid}
                            isLoading={isSubmitting}
                            onClick={() => {
                                analyticsTrack(DOWNLOAD_CLUSTER_REGISTRATION_SECRET);
                                return submitForm();
                            }}
                        >
                            Download
                        </Button>
                        <Button variant="link" isDisabled={isSubmitting} onClick={goBack}>
                            Cancel
                        </Button>
                    </ActionGroup>
                </Flex>
            </PageSection>
        </>
    );
}

export default ClusterRegistrationSecretForm;

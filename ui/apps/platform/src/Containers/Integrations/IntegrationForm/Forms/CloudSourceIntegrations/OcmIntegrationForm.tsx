import React, { ReactElement } from 'react';
import * as yup from 'yup';
import { Button, Checkbox, Form, PageSection, TextInput } from '@patternfly/react-core';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import { CloudSourceIntegration } from 'services/CloudSourceService';
import merge from 'lodash/merge';
import IntegrationHelpIcon from '../Components/IntegrationHelpIcon';
import IntegrationFormActions from '../../IntegrationFormActions';
import useIntegrationForm from '../../useIntegrationForm';
import { IntegrationFormProps } from '../../integrationFormTypes';

function testTokenValue(value, context: yup.TestContext): boolean {
    const requireSecretField =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        !!context?.from[2]?.value?.updateCredentials;
    const clientIdEntered =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        !!context?.from[2]?.value?.cloudSource?.credentials?.clientId?.trim();
    const clientSecretEntered =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        !!context?.from[2]?.value?.cloudSource?.credentials?.clientSecret?.trim();

    if (!requireSecretField || clientIdEntered || clientSecretEntered) {
        return true;
    }
    return !!value?.trim();
}

function testClientValue(value, context: yup.TestContext): boolean {
    const requireSecretField =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        !!context?.from[2]?.value?.updateCredentials;
    const tokenEntered =
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        !!context?.from[2]?.value?.cloudSource?.credentials?.secret?.trim();

    if (!requireSecretField || tokenEntered) {
        return true;
    }
    return !!value?.trim();
}

export const validationSchema = yup.object().shape({
    cloudSource: yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        type: yup.string().matches(/TYPE_OCM/),
        credentials: yup.object().shape({
            secret: yup.string().test('secret-test', 'Token is required', testTokenValue),
            clientId: yup.string().test('client-id-test', 'Client ID is required', testClientValue),
            clientSecret: yup
                .string()
                .test('client-secret-test', 'Client secret is required', testClientValue),
        }),
        ocm: yup.object().shape({
            endpoint: yup.string().trim().required('Endpoint is required'),
        }),
        skipTestIntegration: yup.bool(),
    }),
    updatePassword: yup.bool(),
});

export type CloudSourceIntegrationFormValues = {
    cloudSource: CloudSourceIntegration;
    updateCredentials: boolean;
};
export const defaultValues: CloudSourceIntegrationFormValues = {
    cloudSource: {
        id: '',
        name: '',
        type: 'TYPE_OCM',
        credentials: {
            secret: '',
            clientId: '',
            clientSecret: '',
        },
        skipTestIntegration: true,
        ocm: {
            endpoint: 'https://api.openshift.com',
        },
    },
    updateCredentials: true,
};

function OcmIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<CloudSourceIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.cloudSource = merge({}, formInitialValues.cloudSource, initialValues);
        formInitialValues.cloudSource.credentials.secret = '';
        formInitialValues.cloudSource.credentials.clientId = '';
        formInitialValues.cloudSource.credentials.clientSecret = '';
        formInitialValues.updateCredentials = false;
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
    } = useIntegrationForm<CloudSourceIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });

    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('cloudSource.credentials.secret', '');
        setFieldValue('cloudSource.credentials.clientId', '');
        setFieldValue('cloudSource.credentials.clientSecret', '');
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
                        fieldId="cloudSource.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="cloudSource.name"
                            value={values.cloudSource.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Endpoint"
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="OpenShift Cluster Manager endpoint"
                                helpText={
                                    <div>
                                        The API endpoint under which OpenShift Cluster Manager is
                                        available. Most users will not need to change the preset
                                        value.
                                    </div>
                                }
                                ariaLabel="Help for endpoint"
                            />
                        }
                        fieldId="cloudSource.ocm.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="cloudSource.ocm.endpoint"
                            name="cloudSource.ocm.endpoint"
                            value={values.cloudSource.ocm?.endpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
                        <FormLabelGroup
                            fieldId="updateCredentials"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            errors={errors}
                        >
                            <Checkbox
                                label="Update stored credentials"
                                id="updateCredentials"
                                isChecked={values.updateCredentials}
                                onChange={(event, value) => onUpdateCredentialsChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired={values.updateCredentials}
                        label="Client ID"
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Service account client ID"
                                helpText={
                                    <div>
                                        Client identifier for a{' '}
                                        <Button variant="link" isInline>
                                            <a
                                                href="https://console.redhat.com/iam/service-accounts"
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                Red Hat service account
                                            </a>
                                        </Button>
                                        . The service account must belong to a group with the{' '}
                                        <strong>Subscription Viewer</strong> role. See{' '}
                                        <Button variant="link" isInline>
                                            <a
                                                href="https://docs.redhat.com/en/documentation/openshift_cluster_manager/1-latest/html-single/managing_clusters/index#assembly-user-management-ocm"
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                Configuring access to clusters in OpenShift Cluster
                                                Manager
                                            </a>
                                        </Button>{' '}
                                        for more information.
                                        <br />
                                        <br />
                                        <em>
                                            Service accounts are the preferred authentication method
                                            over the deprecated API token.
                                        </em>
                                    </div>
                                }
                                ariaLabel="Help for client ID"
                            />
                        }
                        fieldId="cloudSource.credentials.clientId"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={
                                values.updateCredentials && !values.cloudSource.credentials.secret
                            }
                            type="text"
                            id={`cloudSource.credentials.clientId`}
                            value={values.cloudSource.credentials.clientId}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updateCredentials}
                            placeholder={
                                values.updateCredentials
                                    ? ''
                                    : 'Currently-stored client ID will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired={values.updateCredentials}
                        label="Client secret"
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="Service account client secret"
                                helpText={
                                    <div>
                                        Client secret for a{' '}
                                        <Button variant="link" isInline>
                                            <a
                                                href="https://console.redhat.com/iam/service-accounts"
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                Red Hat service account
                                            </a>
                                        </Button>
                                        . The service account must belong to a group with the{' '}
                                        <strong>Subscription Viewer</strong> role. See{' '}
                                        <Button variant="link" isInline>
                                            <a
                                                href="https://docs.redhat.com/en/documentation/openshift_cluster_manager/1-latest/html-single/managing_clusters/index#assembly-user-management-ocm"
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                Configuring access to clusters in OpenShift Cluster
                                                Manager
                                            </a>
                                        </Button>{' '}
                                        for more information.
                                        <br />
                                        <br />
                                        <em>
                                            Service accounts are the preferred authentication method
                                            over the deprecated API token.
                                        </em>
                                    </div>
                                }
                                ariaLabel="Help for client secret"
                            />
                        }
                        fieldId="cloudSource.credentials.clientSecret"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={
                                values.updateCredentials && !values.cloudSource.credentials.secret
                            }
                            type="password"
                            id={`cloudSource.credentials.clientSecret`}
                            value={values.cloudSource.credentials.clientSecret}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updateCredentials}
                            placeholder={
                                values.updateCredentials
                                    ? ''
                                    : 'Currently-stored client secret will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired={values.updateCredentials}
                        label="API token (deprecated)"
                        labelIcon={
                            <IntegrationHelpIcon
                                helpTitle="API token"
                                helpText={
                                    <div>
                                        <Button variant="link" isInline>
                                            <a
                                                href="https://console.redhat.com/openshift/token"
                                                target="_blank"
                                                rel="noreferrer"
                                            >
                                                OpenShift Cluster Manager offline token
                                            </a>
                                        </Button>
                                        .
                                        <br />
                                        <br />
                                        <em>
                                            Service accounts are the preferred authentication method
                                            over the deprecated API token.
                                        </em>
                                    </div>
                                }
                                ariaLabel="Help for API token"
                            />
                        }
                        fieldId="cloudSource.credentials.secret"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={
                                values.updateCredentials &&
                                !values.cloudSource.credentials.clientId &&
                                !values.cloudSource.credentials.clientSecret
                            }
                            type="password"
                            id={`cloudSource.credentials.secret`}
                            value={values.cloudSource.credentials.secret}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updateCredentials}
                            placeholder={
                                values.updateCredentials
                                    ? ''
                                    : 'Currently-stored token will be used.'
                            }
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

export default OcmIntegrationForm;

import React, { useState, ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    Checkbox,
    Form,
    PageSection,
    Text,
    TextArea,
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import useMetadata from 'hooks/useMetadata';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import { getVersionedDocs } from 'utils/versioning';
import IntegrationHelpIcon from './Components/IntegrationHelpIcon';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type MicrosoftSentinel = {
    microsoftSentinel: {
        logIngestionEndpoint: string;
        directoryTenantId: string;
        applicationClientId: string;
        clientCertAuthConfig: {
            clientCert: string;
            privateKey: string;
        };
        secret: string;
        alertDcrConfig: {
            dataCollectionRuleId: string;
            streamName: string;
            enabled: boolean;
        };
        auditLogDcrConfig: {
            dataCollectionRuleId: string;
            streamName: string;
            enabled: boolean;
        };
        wifEnabled: boolean;
    };
    type: 'microsoftSentinel';
} & NotifierIntegrationBase;

export type MicrosoftSentinelFormValues = {
    notifier: MicrosoftSentinel;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('A Microsoft Sentinel name is required'),
        microsoftSentinel: yup.object().shape({
            logIngestionEndpoint: yup
                .string()
                .trim()
                .required('A log ingestion endpoint is required'),
            directoryTenantId: yup.string().trim().required('A directory tenant ID is required'),
            applicationClientId: yup.string().trim(),
            secret: yup.string().trim(),
            clientCertAuthConfig: yup.object().shape({
                clientCert: yup.string().trim(),
                privateKey: yup.string().trim(),
            }),
            wifEnabled: yup.boolean(),
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: MicrosoftSentinelFormValues = {
    notifier: {
        id: '',
        name: '',
        microsoftSentinel: {
            logIngestionEndpoint: '',
            directoryTenantId: '',
            applicationClientId: '',
            clientCertAuthConfig: {
                clientCert: '',
                privateKey: '',
            },
            secret: '',
            alertDcrConfig: {
                dataCollectionRuleId: '',
                streamName: '',
                enabled: false,
            },
            auditLogDcrConfig: {
                dataCollectionRuleId: '',
                streamName: '',
                enabled: false,
            },
            wifEnabled: false,
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'microsoftSentinel',
    },
    updatePassword: true,
};

function MicrosoftSentinelForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<MicrosoftSentinel>): ReactElement {
    const formInitialValues = structuredClone(defaultValues);

    if (initialValues) {
        merge(formInitialValues.notifier, initialValues);

        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.microsoftSentinel.secret = '';
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
    } = useIntegrationForm<MicrosoftSentinelFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { version } = useMetadata();
    const { isCreating } = usePageState();
    const [selectedAuthMethod, setSelectedAuthMethod] = useState((): string => {
        if (values.notifier.microsoftSentinel.wifEnabled) {
            return 'use-workload-identity';
        }
        if (values.notifier.microsoftSentinel.clientCertAuthConfig.clientCert) {
            return 'use-client-cert';
        }
        return 'use-secret';
    });

    function onChange(
        value: string | boolean,
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) {
        return setFieldValue(event.currentTarget.id, value);
    }

    function onUpdateAuthMethod(event) {
        const { id } = event.currentTarget;

        setSelectedAuthMethod(id);
    }

    function onUpdateCredentialsChange(
        value: string | boolean,
        event: React.FormEvent<HTMLInputElement>
    ) {
        setFieldValue('notifier.microsoftSentinel.secret', '');
        setFieldValue('notifier.microsoftSentinel.clientCertAuthConfig.privateKey', '');
        return setFieldValue(event.currentTarget.id, value);
    }

    function preHook(callback: () => void) {
        // use only the auth method selected by the user
        if (selectedAuthMethod === 'use-secret') {
            setFieldValue('notifier.microsoftSentinel.clientCertAuthConfig.clientCert', '');
            setFieldValue('notifier.microsoftSentinel.clientCertAuthConfig.privateKey', '');
            setFieldValue('notifier.microsoftSentinel.wifEnabled', false);
        }
        if (selectedAuthMethod === 'use-client-cert') {
            setFieldValue('notifier.microsoftSentinel.secret', '');
            setFieldValue('notifier.microsoftSentinel.wifEnabled', false);
        }
        if (selectedAuthMethod === 'use-workload-identity') {
            if (!isCreating) {
                setFieldValue('updatePassword', true);
            }
            setFieldValue('notifier.microsoftSentinel.applicationClientId', '');
            setFieldValue('notifier.microsoftSentinel.secret', '');
            setFieldValue('notifier.microsoftSentinel.clientCertAuthConfig.clientCert', '');
            setFieldValue('notifier.microsoftSentinel.clientCertAuthConfig.privateKey', '');
            setFieldValue('notifier.microsoftSentinel.wifEnabled', true);
        }

        callback();
    }

    return (
        <>
            <PageSection
                variant="light"
                isFilled
                hasOverflowScroll
                className="microsoft-sentinel-form"
                aria-label="Microsoft Sentinel Form"
            >
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="notifier.name"
                        touched={touched}
                        helperText="(example, Microsoft Integration)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.name"
                            value={values.notifier.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Log ingestion endpoint"
                        isRequired
                        fieldId="notifier.microsoftSentinel.logIngestionEndpoint"
                        touched={touched}
                        helperText="(example, https://example-sentinel-ou812.eastus-1.ingest.monitor.azure.com)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.microsoftSentinel.logIngestionEndpoint"
                            value={values.notifier.microsoftSentinel.logIngestionEndpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Directory tenant ID"
                        isRequired
                        fieldId="notifier.microsoftSentinel.directoryTenantId"
                        touched={touched}
                        helperText="(example, 1234abce-1234-abcd-1234-abcd1234abcd)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.microsoftSentinel.directoryTenantId"
                            value={values.notifier.microsoftSentinel.directoryTenantId}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <Card isFlat>
                        <CardTitle>Authentication method</CardTitle>
                        <CardBody>
                            <ToggleGroup
                                aria-label="Authentication method selection"
                                className="pf-v5-u-pb-md"
                            >
                                <ToggleGroupItem
                                    text="Use secret"
                                    buttonId="use-secret"
                                    isSelected={selectedAuthMethod === 'use-secret'}
                                    onChange={onUpdateAuthMethod}
                                    isDisabled={!isEditable}
                                />
                                <ToggleGroupItem
                                    text="Use client certificate"
                                    buttonId="use-client-cert"
                                    isSelected={selectedAuthMethod === 'use-client-cert'}
                                    onChange={onUpdateAuthMethod}
                                    isDisabled={!isEditable}
                                />
                                <ToggleGroupItem
                                    text="Use workload identity"
                                    buttonId="use-workload-identity"
                                    isSelected={selectedAuthMethod === 'use-workload-identity'}
                                    onChange={onUpdateAuthMethod}
                                    isDisabled={!isEditable}
                                />
                            </ToggleGroup>
                            {!isCreating &&
                                isEditable &&
                                selectedAuthMethod !== 'use-workload-identity' && (
                                    <FormLabelGroup
                                        label=""
                                        fieldId="updatePassword"
                                        helperText="Enable this option to replace currently stored credentials (if any)"
                                        errors={errors}
                                    >
                                        <Checkbox
                                            label="Update authentication"
                                            id="updatePassword"
                                            isChecked={values.updatePassword}
                                            onChange={(event, value) =>
                                                onUpdateCredentialsChange(value, event)
                                            }
                                            onBlur={handleBlur}
                                            isDisabled={!isEditable}
                                        />
                                    </FormLabelGroup>
                                )}
                            {selectedAuthMethod !== 'use-workload-identity' && (
                                <FormLabelGroup
                                    label="Application client ID"
                                    isRequired={values.updatePassword}
                                    fieldId="notifier.microsoftSentinel.applicationClientId"
                                    touched={touched}
                                    helperText="(example, abcd1234-abcd-1234-abcd-1234abce1234)"
                                    errors={errors}
                                >
                                    <TextInput
                                        isRequired
                                        type="text"
                                        id="notifier.microsoftSentinel.applicationClientId"
                                        value={
                                            values.notifier.microsoftSentinel.applicationClientId
                                        }
                                        onChange={(event, value) => onChange(value, event)}
                                        onBlur={handleBlur}
                                        isDisabled={!isEditable}
                                    />
                                </FormLabelGroup>
                            )}
                            {selectedAuthMethod === 'use-secret' && (
                                <FormLabelGroup
                                    label="Secret"
                                    isRequired={values.updatePassword}
                                    fieldId="notifier.microsoftSentinel.secret"
                                    touched={touched}
                                    errors={errors}
                                >
                                    <TextInput
                                        isRequired={values.updatePassword}
                                        type="password"
                                        id="notifier.microsoftSentinel.secret"
                                        value={values.notifier.microsoftSentinel.secret}
                                        onChange={(event, value) => onChange(value, event)}
                                        onBlur={handleBlur}
                                        isDisabled={!isEditable || !values.updatePassword}
                                        placeholder={
                                            values.updatePassword
                                                ? ''
                                                : 'Currently-stored secret will be used.'
                                        }
                                    />
                                </FormLabelGroup>
                            )}
                            {selectedAuthMethod === 'use-client-cert' && (
                                <>
                                    <FormLabelGroup
                                        isRequired={selectedAuthMethod === 'use-client-cert'}
                                        label="Client certificate"
                                        fieldId="notifier.microsoftSentinel.clientCertAuthConfig.clientCert"
                                        touched={touched}
                                        errors={errors}
                                    >
                                        <TextArea
                                            autoResize
                                            resizeOrientation="vertical"
                                            isRequired
                                            type="text"
                                            id="notifier.microsoftSentinel.clientCertAuthConfig.clientCert"
                                            value={
                                                values.notifier.microsoftSentinel
                                                    .clientCertAuthConfig.clientCert
                                            }
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={!isEditable}
                                        />
                                    </FormLabelGroup>
                                    <FormLabelGroup
                                        isRequired={values.updatePassword}
                                        label="Private key"
                                        fieldId="notifier.microsoftSentinel.clientCertAuthConfig.privateKey"
                                        touched={touched}
                                        errors={errors}
                                    >
                                        <TextArea
                                            autoResize
                                            resizeOrientation="vertical"
                                            isRequired
                                            type="text"
                                            id="notifier.microsoftSentinel.clientCertAuthConfig.privateKey"
                                            value={
                                                values.notifier.microsoftSentinel
                                                    .clientCertAuthConfig.privateKey
                                            }
                                            onChange={(event, value) => onChange(value, event)}
                                            onBlur={handleBlur}
                                            isDisabled={!isEditable}
                                        />
                                    </FormLabelGroup>
                                </>
                            )}
                            {selectedAuthMethod === 'use-workload-identity' && (
                                <FormLabelGroup
                                    label="Short-lived tokens"
                                    labelIcon={
                                        <IntegrationHelpIcon
                                            helpTitle="Use workload identity"
                                            helpText={
                                                <>
                                                    <Text>
                                                        Enables authentication with short-lived
                                                        tokens using Azure managed identities or
                                                        Azure workload identities.
                                                    </Text>
                                                    <Text>
                                                        For more information, see{' '}
                                                        <ExternalLink>
                                                            <a
                                                                href={getVersionedDocs(
                                                                    version,
                                                                    'integrating/integrate-using-short-lived-tokens'
                                                                )}
                                                                target="_blank"
                                                                rel="noopener noreferrer"
                                                            >
                                                                RHACS documentation
                                                            </a>
                                                        </ExternalLink>
                                                    </Text>
                                                </>
                                            }
                                            ariaLabel="Help for short-lived tokens"
                                        />
                                    }
                                    helperText="Enabling short-lived tokens removes any existing credentials from this integration"
                                    fieldId="notifier.microsoftSentinel.wifEnabled"
                                    touched={touched}
                                    errors={errors}
                                >
                                    <Checkbox
                                        isRequired
                                        checked
                                        label="Use workload identity"
                                        id="notifier.microsoftSentinel.wifEnabled"
                                        onBlur={handleBlur}
                                        isDisabled={!isEditable}
                                    />
                                </FormLabelGroup>
                            )}
                        </CardBody>
                    </Card>
                    <Card isFlat>
                        <CardTitle>Alert data collection rule configuration</CardTitle>
                        <CardBody>
                            <FormLabelGroup
                                label=""
                                fieldId="notifier.microsoftSentinel.alertDcrConfig.enabled"
                                errors={errors}
                            >
                                <Checkbox
                                    label="Enable alert DCR"
                                    id="notifier.microsoftSentinel.alertDcrConfig.enabled"
                                    isChecked={
                                        values.notifier.microsoftSentinel.alertDcrConfig.enabled
                                    }
                                    onChange={(event, isChecked) => onChange(isChecked, event)}
                                    onBlur={handleBlur}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Alert data collection rule stream name"
                                fieldId="notifier.microsoftSentinel.alertDcrConfig.streamName"
                                touched={touched}
                                helperText="(example, your-custom-sentinel-stream-0123456789)"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="text"
                                    id="notifier.microsoftSentinel.alertDcrConfig.streamName"
                                    value={
                                        values.notifier.microsoftSentinel.alertDcrConfig.streamName
                                    }
                                    onChange={(event, value) => onChange(value, event)}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Alert data collection rule ID"
                                fieldId="notifier.microsoftSentinel.alertDcrConfig.dataCollectionRuleId"
                                touched={touched}
                                helperText="(example, dcr-1234567890abcdef1234567890abcedf)"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="text"
                                    id="notifier.microsoftSentinel.alertDcrConfig.dataCollectionRuleId"
                                    value={
                                        values.notifier.microsoftSentinel.alertDcrConfig
                                            .dataCollectionRuleId
                                    }
                                    onChange={(event, value) => onChange(value, event)}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </CardBody>
                    </Card>
                    <Card isFlat>
                        <CardTitle>Audit data collection rule configuration</CardTitle>
                        <CardBody>
                            <FormLabelGroup
                                label=""
                                fieldId="notifier.microsoftSentinel.auditLogDcrConfig.enabled"
                                errors={errors}
                            >
                                <Checkbox
                                    label="Enable audit log DCR"
                                    id="notifier.microsoftSentinel.auditLogDcrConfig.enabled"
                                    isChecked={
                                        values.notifier.microsoftSentinel.auditLogDcrConfig.enabled
                                    }
                                    onChange={(event, isChecked) => onChange(isChecked, event)}
                                    onBlur={handleBlur}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Audit data collection rule stream name"
                                fieldId="notifier.microsoftSentinel.auditLogDcrConfig.streamName"
                                touched={touched}
                                helperText="(example, your-custom-sentinel-stream-0123456789)"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="text"
                                    id="notifier.microsoftSentinel.auditLogDcrConfig.streamName"
                                    value={
                                        values.notifier.microsoftSentinel.auditLogDcrConfig
                                            .streamName
                                    }
                                    onChange={(event, value) => onChange(value, event)}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Audit data collection rule ID"
                                fieldId="notifier.microsoftSentinel.auditLogDcrConfig.dataCollectionRuleId"
                                touched={touched}
                                helperText="(example, dcr-1234567890abcdef1234567890abcedf)"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="text"
                                    id="notifier.microsoftSentinel.auditLogDcrConfig.dataCollectionRuleId"
                                    value={
                                        values.notifier.microsoftSentinel.auditLogDcrConfig
                                            .dataCollectionRuleId
                                    }
                                    onChange={(event, value) => onChange(value, event)}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </CardBody>
                    </Card>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={() => preHook(onSave)}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={() => preHook(onTest)}
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

export default MicrosoftSentinelForm;

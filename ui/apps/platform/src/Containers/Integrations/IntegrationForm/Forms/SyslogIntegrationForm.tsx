import React, { ReactElement } from 'react';
import {
    TextInput,
    PageSection,
    Form,
    FormSelect,
    FormSelectOption,
    Switch,
} from '@patternfly/react-core';
import * as yup from 'yup';

import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';

export type SyslogIntegration = {
    id?: string;
    name: string;
    syslog: {
        localFacility: string;
        tcpConfig: {
            hostname: string;
            port: number;
            useTls: boolean;
            skipTlsVerify: boolean;
        };
    };
    uiEndpoint: string;
    type: 'syslog';
    enabled: boolean;
};

export const validationSchema = yup.object().shape({
    name: yup.string().required('Integration name is required'),
    syslog: yup.object().shape({
        localFacility: yup.string().required('Logging facility is required'),
        tcpConfig: yup.object().shape({
            hostname: yup.string().required('Receiver host is required'),
            port: yup
                .number()
                .required('Receiver port is required')
                .test(
                    'receiver-port-test',
                    'Receiver port must be between 1 and 65535',
                    (value = 0) => {
                        return value >= 1 && value <= 65535;
                    }
                ),
            useTls: yup.bool(),
            skipTlsVerify: yup.bool(),
        }),
    }),
    uiEndpoint: yup.string(),
    type: yup.string().matches(/syslog/),
    enabled: yup.bool(),
});

export const defaultValues: SyslogIntegration = {
    syslog: {
        localFacility: '',
        tcpConfig: {
            hostname: '',
            port: 514,
            useTls: false,
            skipTlsVerify: false,
        },
    },
    name: '',
    uiEndpoint: window.location.origin,
    type: 'syslog',
    enabled: true,
};

function SyslogIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<SyslogIntegration>): ReactElement {
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
    } = useIntegrationForm<SyslogIntegration, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
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
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Logging facility"
                        fieldId="syslog.localFacility"
                        touched={touched}
                        errors={errors}
                    >
                        <FormSelect
                            isRequired
                            id="syslog.localFacility"
                            value={values.syslog.localFacility}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        >
                            <FormSelectOption key={0} label="Choose one..." value="" isDisabled />
                            <FormSelectOption key={1} label="local0" value="LOCAL0" />
                            <FormSelectOption key={2} label="local1" value="LOCAL1" />
                            <FormSelectOption key={3} label="local2" value="LOCAL2" />
                            <FormSelectOption key={4} label="local3" value="LOCAL3" />
                            <FormSelectOption key={5} label="local4" value="LOCAL4" />
                            <FormSelectOption key={6} label="local5" value="LOCAL5" />
                            <FormSelectOption key={7} label="local6" value="LOCAL6" />
                            <FormSelectOption key={8} label="local7" value="LOCAL7" />
                        </FormSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Receiver host"
                        fieldId="syslog.tcpConfig.hostname"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="syslog.tcpConfig.hostname"
                            value={values.syslog.tcpConfig.hostname}
                            placeholder="example, host.example.com)"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Receiver port"
                        fieldId="syslog.tcpConfig.port"
                        touched={touched}
                        errors={errors}
                        helperText="A port number between 1 and 65535"
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="syslog.tcpConfig.port"
                            value={values.syslog.tcpConfig.port}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use TLS"
                        fieldId="syslog.tcpConfig.useTls"
                        errors={errors}
                    >
                        <Switch
                            id="syslog.tcpConfig.useTls"
                            name="syslog.tcpConfig.useTls"
                            aria-label="use tls"
                            isChecked={values.syslog.tcpConfig.useTls}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Disable TLS certificate validation (insecure)"
                        fieldId="syslog.tcpConfig.skipTlsVerify"
                        errors={errors}
                    >
                        <Switch
                            id="syslog.tcpConfig.skipTlsVerify"
                            name="syslog.tcpConfig.skipTlsVerify"
                            aria-label="disable tls certificate validation"
                            isChecked={values.syslog.tcpConfig.skipTlsVerify}
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

export default SyslogIntegrationForm;

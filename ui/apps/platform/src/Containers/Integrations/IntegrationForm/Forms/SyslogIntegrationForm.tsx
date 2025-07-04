/* eslint-disable prettier/prettier */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import {
    Button,
    Checkbox,
    Flex,
    FlexItem,
    PageSection,
    Form,
    FormSection,
    FormSelect,
    FormSelectOption,
    TextInput,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import { PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';
import merge from 'lodash/merge';

import { SyslogNotifierIntegration as SyslogIntegration } from 'types/notifier.proto';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

import './SyslogIntegrationForm.css';

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
        maxMessageSize: yup
            .number()
            .required('Message size is required')
            .test(
                'message-size-test',
                'Message size must be between 0 and 1048576',
                (value) => value >= 0 && value <= 1048576
            ),
    }),
    uiEndpoint: yup.string(),
    type: yup.string().matches(/syslog/),
});

export const defaultValues: SyslogIntegration = {
    id: '',
    name: '',
    syslog: {
        messageFormat: 'CEF',
        localFacility: undefined,
        extraFields: [],
        maxMessageSize: 0,
        tcpConfig: {
            hostname: '',
            port: 514,
            useTls: false,
            skipTlsVerify: false,
        },
    },
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
    type: 'syslog',
};

function SyslogIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<SyslogIntegration>): ReactElement {
    const formInitialValues: SyslogIntegration = merge({}, defaultValues, initialValues);
    const formik = useIntegrationForm<SyslogIntegration>({
        initialValues: formInitialValues,
        validationSchema,
    });
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
    } = formik;

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll aria-label="Syslog Form Section">
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormikProvider value={formik}>
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
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            >
                                <FormSelectOption
                                    key={0}
                                    label="Choose one..."
                                    value=""
                                    isDisabled
                                />
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
                                placeholder="(example, host.example.com)"
                                onChange={(event, value) => onChange(value, event)}
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
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            isRequired
                            label="Maximum message size"
                            fieldId="syslog.maxMessageSize"
                            touched={touched}
                            errors={errors}
                            helperText="The number of bytes between 0 and 1048576 to chunk messages in. A 0 means messages will not be chunked."
                        >
                            <TextInput
                                isRequired
                                type="number"
                                id="syslog.maxMessageSize"
                                value={values.syslog.maxMessageSize}
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Message Format"
                            isRequired
                            fieldId="messageFormat"
                            touched={touched}
                            errors={errors}
                            helperText="For new integrations, choose CEF. If you have an existing integration that relies on the old behavior, leave Legacy selected."
                        >
                            <ToggleGroup id="messageFormat" areAllGroupsDisabled={!isEditable}>
                                        <ToggleGroupItem
                                            // The HTML ID and custom CSS rule are required to make the shorter option similar in size to the longer option
                                            // because PatternFly does not allow the inner width of the toggle button to be expanded easily
                                            // (setting a min-witch on Toggle Item just adds space to the right of the outlined button)
                                            id="CEF-option"
                                            key='CEF'
                                            text="CEF"
                                            isSelected={values.syslog.messageFormat === 'CEF'}
                                            onChange={() =>
                                                setFieldValue('syslog.messageFormat', 'CEF')
                                            }
                                        />
                                        <ToggleGroupItem
                                            key='LEGACY'
                                            text="CEF (legacy field order)"
                                            isSelected={values.syslog.messageFormat === 'LEGACY' || !values.syslog.messageFormat}
                                            onChange={() =>
                                                setFieldValue('syslog.messageFormat', 'LEGACY')
                                            }
                                        />
                            </ToggleGroup>
                        </FormLabelGroup>
                        <FormLabelGroup fieldId="syslog.tcpConfig.useTls" errors={errors}>
                            <Checkbox
                                label="Use TLS"
                                id="syslog.tcpConfig.useTls"
                                isChecked={values.syslog.tcpConfig.useTls}
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup fieldId="syslog.tcpConfig.skipTlsVerify" errors={errors}>
                            <Checkbox
                                label="Disable TLS certificate validation (insecure)"
                                id="syslog.tcpConfig.skipTlsVerify"
                                isChecked={values.syslog.tcpConfig.skipTlsVerify}
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormSection title="Extra Fields" titleElement="h3" className="pf-v5-u-mt-0">
                            <FieldArray
                                name="syslog.extraFields"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.syslog.extraFields.length === 0 && (
                                            <p>No custom extra fields defined</p>
                                        )}
                                        {values.syslog.extraFields.length > 0 &&
                                            values.syslog.extraFields.map(
                                                (_extraField, index: number) => (
                                                    <Flex key={`extraField_${index}`}>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Key"
                                                                fieldId={`syslog.extraFields[${index}].key`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`syslog.extraFields[${index}].key`}
                                                                    value={
                                                                        values.syslog
                                                                            .extraFields[`${index}`]
                                                                            .key
                                                                    }
                                                                    onChange={(event, value) => onChange(value, event)}
                                                                    onBlur={handleBlur}
                                                                    isDisabled={!isEditable}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Value"
                                                                fieldId={`syslog.extraFields[${index}].value`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`syslog.extraFields[${index}].value`}
                                                                    value={
                                                                        values.syslog
                                                                            .extraFields[`${index}`]
                                                                            .value
                                                                    }
                                                                    onChange={(event, value) => onChange(value, event)}
                                                                    onBlur={handleBlur}
                                                                    isDisabled={!isEditable}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                        {isEditable && (
                                                            <FlexItem>
                                                                <Button
                                                                    variant="plain"
                                                                    aria-label="Delete extra field key/value pair"
                                                                    style={{
                                                                        transform:
                                                                            'translate(0, 42px)',
                                                                    }}
                                                                    onClick={() =>
                                                                        arrayHelpers.remove(index)
                                                                    }
                                                                >
                                                                    <TrashIcon />
                                                                </Button>
                                                            </FlexItem>
                                                        )}
                                                    </Flex>
                                                )
                                            )}
                                        {isEditable && (
                                            <Flex>
                                                <FlexItem>
                                                    <Button
                                                        variant="link"
                                                        isInline
                                                        icon={
                                                            <PlusCircleIcon className="pf-v5-u-mr-sm" />
                                                        }
                                                        onClick={() =>
                                                            arrayHelpers.push({
                                                                key: '',
                                                                value: '',
                                                            })
                                                        }
                                                    >
                                                        Add new extra field
                                                    </Button>
                                                </FlexItem>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </FormSection>
                    </FormikProvider>
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

export default SyslogIntegrationForm;

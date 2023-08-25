/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import {
    Checkbox,
    Flex,
    FlexItem,
    Form,
    FormFieldGroupExpandable,
    FormFieldGroupHeader,
    PageSection,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import merge from 'lodash/merge';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type JiraIntegration = {
    jira: {
        username: string;
        password: string;
        issueType: string;
        url: string;
        priorityMappings: {
            severity: string;
            priorityName: string;
        }[];
        defaultFieldsJson: string;
    };
    type: 'jira';
} & NotifierIntegrationBase;

export type JiraIntegrationFormValues = {
    notifier: JiraIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('Name is required'),
        jira: yup.object().shape({
            username: yup.string().trim().required('Username is required'),
            password: yup
                .string()
                .test(
                    'password-token-test',
                    'Password or API token is required',
                    (value, context: yup.TestContext) => {
                        const requireHttpTokenField =
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            context?.from[2]?.value?.updatePassword || false;

                        if (!requireHttpTokenField) {
                            return true;
                        }

                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
            issueType: yup.string().trim().required('Issue type is required'),
            url: yup.string().trim().required('Jira URL is required'),
        }),
        labelDefault: yup.string().trim().required('A default project is required'),
        labelKey: yup.string().trim(),
    }),
    updatePassword: yup.bool(),
});

const defaultSeverities = [
    {
        severity: 'CRITICAL_SEVERITY',
        priorityName: 'P0-Highest',
    },
    {
        severity: 'HIGH_SEVERITY',
        priorityName: 'P1-High',
    },
    {
        severity: 'MEDIUM_SEVERITY',
        priorityName: 'P2-Medium',
    },
    {
        severity: 'LOW_SEVERITY',
        priorityName: 'P3-Low',
    },
];

export const defaultValues: JiraIntegrationFormValues = {
    notifier: {
        id: '',
        name: '',
        jira: {
            username: '',
            password: '',
            issueType: '',
            url: '',
            priorityMappings: defaultSeverities,
            defaultFieldsJson: '',
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'jira',
    },
    updatePassword: true,
};

function JiraIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<JiraIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.notifier = merge({}, defaultValues.notifier, initialValues); // in case properties are missing from initialValues

        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.jira.password = '';

        // Don't assume user wants to change password; that has caused confusing UX.
        formInitialValues.updatePassword = false;
    }
    const formik = useIntegrationForm<JiraIntegrationFormValues>({
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
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('notifier.jira.password', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormikProvider value={formik}>
                        <FormLabelGroup
                            isRequired
                            label="Integration name"
                            fieldId="notifier.name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.name"
                                value={values.notifier.name}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Username"
                            isRequired
                            fieldId="notifier.jira.username"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.jira.username"
                                value={values.notifier.jira.username}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        {!isCreating && isEditable && (
                            <FormLabelGroup
                                label=""
                                fieldId="updatePassword"
                                helperText="Enable this option to replace currently stored credentials (if any)"
                                errors={errors}
                            >
                                <Checkbox
                                    label="Update password"
                                    id="updatePassword"
                                    isChecked={values.updatePassword}
                                    onChange={onUpdateCredentialsChange}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        )}
                        <FormLabelGroup
                            isRequired={values.updatePassword}
                            label="Password or API token"
                            fieldId="notifier.jira.password"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired={values.updatePassword}
                                type="password"
                                id="notifier.jira.password"
                                value={values.notifier.jira.password}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || !values.updatePassword}
                                placeholder={
                                    values.updatePassword
                                        ? ''
                                        : 'Currently-stored password will be used.'
                                }
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Issue type"
                            isRequired
                            fieldId="notifier.jira.issueType"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.jira.issueType"
                                value={values.notifier.jira.issueType}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                                placeholder="Epic, Story, Task, Sub-task, or Bug"
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Jira URL"
                            isRequired
                            fieldId="notifier.jira.url"
                            touched={touched}
                            errors={errors}
                            helperText="example, https://example.atlassian.net"
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.jira.url"
                                value={values.notifier.jira.url}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Default project"
                            isRequired
                            fieldId="notifier.labelDefault"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.labelDefault"
                                value={values.notifier.labelDefault}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Annotation key for project"
                            labelIcon={<AnnotationKeyLabelIcon />}
                            fieldId="notifier.labelKey"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                type="text"
                                id="notifier.labelKey"
                                value={values.notifier.labelKey}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormFieldGroupExpandable
                            isExpanded={false}
                            toggleAriaLabel="Toggle showing Jira priority mappings"
                            header={
                                <FormFieldGroupHeader
                                    titleText={{
                                        text: 'Priority Mappings',
                                        id: 'priority-mappings-id',
                                    }}
                                />
                            }
                        >
                            <FieldArray
                                name="notifier.jira.priorityMappings"
                                render={() => (
                                    <>
                                        {values.notifier.jira.priorityMappings.length === 0 && (
                                            <p>No custom priorityMappings defined</p>
                                        )}
                                        {values.notifier.jira.priorityMappings.length > 0 &&
                                            values.notifier.jira.priorityMappings.map(
                                                (_priorityMapping, index: number) => (
                                                    <Flex key={`header_${index}`}>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label={
                                                                    index === 0 ? 'Severity' : ''
                                                                }
                                                                fieldId={`notifier.jira.priorityMappings[${index}].severity`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    tabIndex={-1}
                                                                    className="pf-u-background-color-200"
                                                                    isReadOnly
                                                                    type="text"
                                                                    id={`notifier.jira.priorityMappings[${index}].severity`}
                                                                    value={
                                                                        values.notifier.jira
                                                                            .priorityMappings[
                                                                            `${index}`
                                                                        ].severity
                                                                    }
                                                                    onChange={onChange}
                                                                    onBlur={handleBlur}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label={
                                                                    index === 0
                                                                        ? 'Priority Name'
                                                                        : ''
                                                                }
                                                                fieldId={`notifier.jira.priorityMappings[${index}].priorityName`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    aria-labelledby={`notifier.jira.priorityMappings[${index}].severity`}
                                                                    type="text"
                                                                    id={`notifier.jira.priorityMappings[${index}].priorityName`}
                                                                    value={
                                                                        values.notifier.jira
                                                                            .priorityMappings[
                                                                            `${index}`
                                                                        ].priorityName
                                                                    }
                                                                    onChange={onChange}
                                                                    onBlur={handleBlur}
                                                                    isDisabled={!isEditable}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                    </Flex>
                                                )
                                            )}
                                    </>
                                )}
                            />
                        </FormFieldGroupExpandable>
                        {/* </FormSection>
                        )} */}
                        <FormLabelGroup
                            label="Default Fields JSON"
                            fieldId="notifier.jira.defaultFieldsJson"
                            touched={touched}
                            errors={errors}
                            helperText="Necessary if Jira has required fields"
                        >
                            <TextArea
                                className="json-input"
                                type="text"
                                id="notifier.jira.defaultFieldsJson"
                                value={values.notifier.jira.defaultFieldsJson}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
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

export default JiraIntegrationForm;

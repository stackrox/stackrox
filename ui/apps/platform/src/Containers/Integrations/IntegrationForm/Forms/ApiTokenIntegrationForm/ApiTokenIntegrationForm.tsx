import React, { ReactElement } from 'react';
import {
    DatePicker,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Form,
    PageSection,
    TextInput,
    yyyyMMddFormat,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';

import * as yup from 'yup';

import { ApiToken } from 'types/apiToken.proto';

import SelectSingle from 'Components/SelectSingle';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import { getDateTime } from 'utils/dateUtils';
import NotFoundMessage from 'Components/NotFoundMessage';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../../useIntegrationForm';
import IntegrationFormActions from '../../IntegrationFormActions';
import ApiTokenFormMessageAlert, { ApiTokenFormResponseMessage } from './ApiTokenFormMessageAlert';
import FormLabelGroup from '../../FormLabelGroup';
import useAllowedRoles from './useFetchRoles';

export type ApiTokenIntegrationFormValues = {
    name: string;
    roles: string[];
    expiration?: string;
};

export type ApiTokenIntegrationFormProps = {
    initialValues: ApiToken | null;
    isEditable?: boolean;
};

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('A token name is required'),
    roles: yup
        .array()
        .of(yup.string().trim())
        .min(1, 'Must have a role selected')
        .required('A role is required'),
    expiration: yup
        .string()
        .test('Future date test', 'Expiration date should not be in the past', (value) => {
            if (!value) {
                return true;
            }
            return new Date(value) > new Date();
        }),
});

export const defaultValues: ApiTokenIntegrationFormValues = {
    name: '',
    roles: [],
};

function ApiTokenIntegrationForm({
    initialValues = null,
    isEditable = false,
}: ApiTokenIntegrationFormProps): ReactElement {
    const formInitialValues = initialValues ? { ...initialValues, defaultValues } : defaultValues;
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
        onCancel,
        message,
    } = useIntegrationForm<ApiTokenIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isEditing, isViewingDetails } = usePageState();
    const { roleNames, isLoading: isRolesLoading } = useAllowedRoles();
    const isGenerated = Boolean((message as ApiTokenFormResponseMessage)?.responseData);

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onRoleChange(id, selection) {
        return setFieldValue(id, [selection]);
    }

    // The edit flow doesn't make sense for API Tokens so we'll show an empty state message here
    if (isEditing) {
        return (
            <NotFoundMessage
                title="This API Token cannot be edited"
                message="Create a new API Token or delete an existing one"
            />
        );
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <div id="form-message-alert" className="pf-v5-u-pb-md">
                    {message && <ApiTokenFormMessageAlert message={message} />}
                </div>

                {isViewingDetails && initialValues ? (
                    <DescriptionList>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Name</DescriptionListTerm>
                            <DescriptionListDescription>
                                {initialValues.name}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Role</DescriptionListTerm>
                            <DescriptionListDescription>
                                {initialValues.roles.join(', ')}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Issued</DescriptionListTerm>
                            <DescriptionListDescription>
                                {getDateTime(initialValues.issuedAt)}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Expiration</DescriptionListTerm>
                            <DescriptionListDescription>
                                {getDateTime(initialValues.expiration)}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Revoked</DescriptionListTerm>
                            <DescriptionListDescription>
                                {initialValues.revoked ? 'Yes' : 'No'}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                ) : (
                    <Form isWidthLimited>
                        <FormLabelGroup
                            label="Token name"
                            isRequired
                            fieldId="name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                type="text"
                                id="name"
                                value={values.name}
                                onChange={(event, value) => onChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || isGenerated}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            isRequired
                            label="Role"
                            fieldId="roles"
                            touched={touched}
                            errors={errors}
                        >
                            <SelectSingle
                                id="roles"
                                value={values.roles[0]}
                                handleSelect={onRoleChange}
                                isDisabled={!isEditable || isRolesLoading || isGenerated}
                                placeholderText="Choose role..."
                            >
                                {roleNames.map((roleName) => {
                                    return (
                                        <SelectOption key={roleName} value={roleName}>
                                            {roleName}
                                        </SelectOption>
                                    );
                                })}
                            </SelectSingle>
                        </FormLabelGroup>
                        {isEditable && !isGenerated && (
                            <FormLabelGroup
                                label="Expiration date"
                                fieldId="expiration"
                                touched={touched}
                                helperText="when left unset, the value defaults to 1 year from the generation date"
                                errors={errors}
                            >
                                <DatePicker
                                    id="expiration"
                                    inputProps={{ id: 'expiration' }}
                                    onBlur={handleBlur}
                                    value={
                                        values.expiration
                                            ? yyyyMMddFormat(new Date(values.expiration))
                                            : ''
                                    }
                                    onChange={(event, value, date) => {
                                        if (date) {
                                            setFieldValue('expiration', date.toISOString());
                                        } else {
                                            setFieldValue('expiration', undefined);
                                        }
                                    }}
                                />
                            </FormLabelGroup>
                        )}
                    </Form>
                )}
            </PageSection>
            {isEditable &&
                (!isGenerated ? (
                    <IntegrationFormActions>
                        <FormSaveButton
                            onSave={onSave}
                            isSubmitting={isSubmitting}
                            isTesting={isTesting}
                            isDisabled={!dirty || !isValid}
                        >
                            Generate
                        </FormSaveButton>
                        <FormCancelButton onCancel={onCancel} isDisabled={isSubmitting}>
                            Cancel
                        </FormCancelButton>
                    </IntegrationFormActions>
                ) : (
                    <IntegrationFormActions>
                        <FormCancelButton onCancel={onCancel} isDisabled={isSubmitting}>
                            Back
                        </FormCancelButton>
                    </IntegrationFormActions>
                ))}
        </>
    );
}

export default ApiTokenIntegrationForm;

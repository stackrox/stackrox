import React, { ReactElement } from 'react';
import {
    TextInput,
    PageSection,
    Form,
    FormSelect,
    FormSelectOption,
    DescriptionList,
    DescriptionListTerm,
    DescriptionListGroup,
    DescriptionListDescription,
} from '@patternfly/react-core';

import * as yup from 'yup';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import { getDateTime } from 'utils/dateUtils';
import NotFoundMessage from 'Components/NotFoundMessage';
import useIntegrationForm from '../../useIntegrationForm';
import IntegrationFormActions from '../../IntegrationFormActions';
import FormCancelButton from '../../FormCancelButton';
import FormSaveButton from '../../FormSaveButton';
import ApiTokenFormMessageAlert, { ApiTokenFormResponseMessage } from './ApiTokenFormMessageAlert';
import FormLabelGroup from '../../FormLabelGroup';
import useFetchRoles from './useFetchRoles';

export type ApiTokenIntegration = {
    expiration: string;
    id: string;
    issuedAt: string;
    name: string;
    revoked: boolean;
    roles: string[];
};

export type ApiTokenIntegrationFormValues = {
    name: string;
    roles: string[];
};

export type ApiTokenIntegrationFormProps = {
    initialValues: ApiTokenIntegration | null;
    isEditable?: boolean;
};

export const validationSchema = yup.object().shape({
    name: yup.string().required('Required'),
    roles: yup.array().of(yup.string()).min(1, 'Must have a role selected').required('Required'),
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
        errors,
        setFieldValue,
        isSubmitting,
        isTesting,
        onSave,
        onCancel,
        message,
    } = useIntegrationForm<ApiTokenIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isEditing, isViewingDetails } = usePageState();
    const { roles, isLoading: isRolesLoading } = useFetchRoles();
    const isGenerated = Boolean(message?.responseData);

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function onRoleChange(value, event) {
        return setFieldValue(event.target.id, [value], false);
    }

    // The edit flow doesn't make sense for API Tokens so we'll show an empty state message here
    if (isEditing) {
        return (
            <NotFoundMessage
                title="This API Token can not be edited"
                message="Create a new API Token or delete an existing one"
            />
        );
    }

    return (
        <>
            {message && (
                <ApiTokenFormMessageAlert message={message as ApiTokenFormResponseMessage} />
            )}
            <PageSection variant="light" isFilled hasOverflowScroll>
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
                            label="Token Name"
                            isRequired
                            fieldId="name"
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="name"
                                name="name"
                                value={values.name}
                                onChange={onChange}
                                isDisabled={!isEditable || isGenerated}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup isRequired label="Role" fieldId="roles" errors={errors}>
                            <FormSelect
                                id="roles"
                                value={values.roles}
                                onChange={onRoleChange}
                                isDisabled={!isEditable || isRolesLoading || isGenerated}
                            >
                                <FormSelectOption label="Choose one..." value="" isDisabled />
                                {roles.map((role) => {
                                    return (
                                        <FormSelectOption
                                            key={role.name}
                                            label={role.name}
                                            value={role.name}
                                        />
                                    );
                                })}
                            </FormSelect>
                        </FormLabelGroup>
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
                            Close
                        </FormCancelButton>
                    </IntegrationFormActions>
                ))}
        </>
    );
}

export default ApiTokenIntegrationForm;

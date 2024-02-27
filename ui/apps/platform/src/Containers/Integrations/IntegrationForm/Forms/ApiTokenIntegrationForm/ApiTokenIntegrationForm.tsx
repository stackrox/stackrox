import React, { ReactElement } from 'react';
import {
    TextInput,
    PageSection,
    Form,
    DescriptionList,
    DescriptionListTerm,
    DescriptionListGroup,
    DescriptionListDescription,
} from '@patternfly/react-core';

import * as yup from 'yup';

import { ApiToken } from 'types/apiToken.proto';

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
import RoleSelector from './RoleSelector';

export type ApiTokenIntegrationFormValues = {
    name: string;
    roles: string[];
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

    const onRoleSelect = (_, selected) => {
        const newSelection = values.roles.find((roleFilter) => roleFilter === selected)
            ? values.roles.filter((roleFilter) => roleFilter !== selected)
            : values.roles.concat(selected);

        return setFieldValue('roles', newSelection);
    };

    const onClearRoleSelections = () => {
        return setFieldValue('roles', []);
    };

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
            <PageSection variant="light" isFilled hasOverflowScroll>
                <div id="form-message-alert" className="pf-u-pb-md">
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
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || isGenerated}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            isRequired
                            label="Roles"
                            fieldId="roles"
                            touched={touched}
                            errors={errors}
                        >
                            <RoleSelector
                                roles={roleNames}
                                selectedRoles={values.roles}
                                isEditable={isEditable}
                                isGenerated={isGenerated}
                                isRolesLoading={isRolesLoading}
                                onRoleSelect={onRoleSelect}
                                onRoleSelectionClear={onClearRoleSelections}
                            />
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

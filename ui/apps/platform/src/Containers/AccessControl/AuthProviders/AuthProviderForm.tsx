import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    FormGroup,
    SelectOption,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { availableAuthProviders } from 'constants/accessControl';

import { AccessControlQueryAction, AuthProvider, Role } from '../accessControlTypes';
import SelectSingle from '../SelectSingle'; // TODO import from where?

export type AuthProviderFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    authProvider: AuthProvider;
    roles: Role[];
    onClickCancel: () => void;
    onClickEdit: () => void;
    submitValues: (values: AuthProvider) => Promise<AuthProvider>;
};

function AuthProviderForm({
    isActionable,
    action,
    authProvider,
    roles,
    onClickCancel,
    onClickEdit,
    submitValues,
}: AuthProviderFormProps): ReactElement {
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [alertSubmit, setAlertSubmit] = useState<ReactElement | null>(null);

    // TODO Why does browser refresh when form is open cause values to be undefined?
    const { dirty, handleChange, isValid, setFieldValue, values } = useFormik({
        initialValues: authProvider,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            // authProvider
            // minimumAccessRole
        }),
    });

    function onChange(_value, event) {
        handleChange(event);
    }

    function onClickSubmit() {
        // TODO submit through Formik, especially to update its initialValue.
        // For example, to make a change, submit, and then make the opposite change.
        setIsSubmitting(true);
        setAlertSubmit(null);
        submitValues(values)
            .catch((error) => {
                setAlertSubmit(
                    <Alert
                        title="Failed to submit auth provider"
                        variant={AlertVariant.danger}
                        isInline
                    >
                        {error.message}
                    </Alert>
                );
            })
            .finally(() => {
                setIsSubmitting(false);
            });
    }

    const hasAction = Boolean(action);
    const isViewing = !hasAction;

    // TODO Minimum access role: replace select with radio button table as in Role form?
    return (
        <Form>
            {isActionable && (
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        {action !== 'create' && (
                            <ToolbarItem spacer={{ default: 'spacerLg' }}>
                                <Button
                                    variant="primary"
                                    onClick={onClickEdit}
                                    isDisabled={action === 'update'}
                                    isSmall
                                >
                                    Edit auth provider
                                </Button>
                            </ToolbarItem>
                        )}
                        {hasAction && (
                            <ToolbarGroup variant="button-group">
                                <ToolbarItem>
                                    <Button
                                        variant="primary"
                                        onClick={onClickSubmit}
                                        isDisabled={!dirty || !isValid || isSubmitting}
                                        isLoading={isSubmitting}
                                        isSmall
                                    >
                                        Submit
                                    </Button>
                                </ToolbarItem>
                                <ToolbarItem>
                                    <Button variant="tertiary" onClick={onClickCancel} isSmall>
                                        Cancel
                                    </Button>
                                </ToolbarItem>
                            </ToolbarGroup>
                        )}
                    </ToolbarContent>
                </Toolbar>
            )}
            {alertSubmit}
            <FormGroup label="Name" fieldId="name" isRequired>
                <TextInput
                    type="text"
                    id="name"
                    value={values.name}
                    onChange={onChange}
                    isDisabled={isViewing}
                    isRequired
                />
            </FormGroup>
            <FormGroup label="Auth provider" fieldId="authProvider" isRequired>
                <SelectSingle
                    id="authProvider"
                    value={values.authProvider}
                    setFieldValue={setFieldValue}
                    isDisabled={isViewing}
                >
                    {availableAuthProviders.map(({ value, label }) => (
                        <SelectOption key={value} value={value}>
                            {label}
                        </SelectOption>
                    ))}
                </SelectSingle>
            </FormGroup>
            <FormGroup label="Minimum access role" fieldId="minimumAccessRole" isRequired>
                <SelectSingle
                    id="minimumAccessRole"
                    value={values.minimumAccessRole}
                    setFieldValue={setFieldValue}
                    isDisabled={isViewing}
                >
                    {roles.map(({ name }) => (
                        <SelectOption key={name} value={name} />
                    ))}
                </SelectSingle>
            </FormGroup>
        </Form>
    );
}

export default AuthProviderForm;

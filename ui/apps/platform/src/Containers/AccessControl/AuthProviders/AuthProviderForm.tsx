import React, { ReactElement, useState } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    FormGroup,
    Grid,
    GridItem,
    SelectOption,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import { availableAuthProviders } from 'constants/accessControl';
import { AuthProvider } from 'services/AuthService';
import { Role } from 'services/RolesService';

import { getInitialAuthProviderValues } from './authProviders.utils';
import { AccessControlQueryAction } from '../accessControlPaths';

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

    const initialValues = getInitialAuthProviderValues(authProvider);

    const { dirty, handleChange, isValid, setFieldValue, values } = useFormik({
        initialValues,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            type: yup.string().required(),
            config: yup.object().when('type', {
                is: 'auth0',
                then: yup.object({
                    issuer: yup.string().required(),
                    client_id: yup.string().required(),
                }),
            }),
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
            <Grid hasGutter>
                <GridItem span={12} lg={6}>
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
                </GridItem>
                <GridItem span={12} lg={6}>
                    <FormGroup label="Auth provider type" fieldId="type" isRequired>
                        <SelectSingle
                            id="type"
                            value={values.type}
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
                </GridItem>
                <GridItem span={12} lg={6}>
                    <FormGroup label="Auth0 tenant" fieldId="name" isRequired>
                        <TextInput
                            type="text"
                            id="config.issuer"
                            value={values.config.issuer || ''}
                            onChange={onChange}
                            isDisabled={isViewing}
                            isRequired
                        />
                    </FormGroup>
                </GridItem>
                <GridItem span={12} lg={6}>
                    <FormGroup label="Client ID" fieldId="name" isRequired>
                        <TextInput
                            type="text"
                            id="config.client_id"
                            value={values.config.client_id || ''}
                            onChange={onChange}
                            isDisabled={isViewing}
                            isRequired
                        />
                    </FormGroup>
                </GridItem>
            </Grid>
            <FormGroup label="Minimum access role" fieldId="minimumAccessRole" isRequired>
                <SelectSingle
                    id="minimumAccessRole"
                    value="" // TODO see getDefaultRoleByAuthProviderId in classic code
                    setFieldValue={setFieldValue}
                    isDisabled={isViewing}
                >
                    {roles.map(({ name }) => (
                        <SelectOption key={name} value={name} />
                    ))}
                </SelectSingle>
            </FormGroup>
            <FormGroup label="Rules" fieldId="rules" />
        </Form>
    );
}

export default AuthProviderForm;

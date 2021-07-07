import React, { ReactElement } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useFormik, FormikProvider } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    Button,
    Form,
    FormGroup,
    FormSection,
    Grid,
    GridItem,
    SelectOption,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle'; // TODO import from where?
import { availableAuthProviders } from 'constants/accessControl';
import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { AuthProvider } from 'services/AuthService';

import ConfigurationFormFields from './ConfigurationFormFields';
import RuleGroups from './RuleGroups';
import {
    getInitialAuthProviderValues,
    transformInitialValues,
    transformValuesBeforeSaving,
    getGroupsByAuthProviderId,
    getDefaultRoleByAuthProviderId,
} from './authProviders.utils';
import { AccessControlQueryAction } from '../accessControlPaths';

export type AuthProviderFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    selectedAuthProvider: AuthProvider;
    onClickCancel: () => void;
    onClickEdit: () => void;
};

const authProviderState = createStructuredSelector({
    roles: selectors.getRoles,
    groups: selectors.getRuleGroups,
    saveAuthProviderError: selectors.getSaveAuthProviderError,
});

function getNewAuthProviderTitle(type) {
    const selectedType = availableAuthProviders.find(({ value }) => value === type);

    return `Add new ${selectedType?.label as string} auth provider`;
}
function AuthProviderForm({
    isActionable,
    action,
    selectedAuthProvider,
    onClickCancel,
    onClickEdit,
}: AuthProviderFormProps): ReactElement {
    const { groups, roles, saveAuthProviderError } = useSelector(authProviderState);
    const dispatch = useDispatch();

    const initialValues = getInitialAuthProviderValues(selectedAuthProvider);
    const filteredGroups = getGroupsByAuthProviderId(groups, selectedAuthProvider.id);
    const defaultRole = getDefaultRoleByAuthProviderId(groups, selectedAuthProvider.id);

    const modifiedInitialValues = {
        ...transformInitialValues(initialValues),
        groups: filteredGroups,
        defaultRole,
    };

    const formik = useFormik({
        initialValues: modifiedInitialValues,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            type: yup.string().required(),
            config: yup
                .object()
                .when('type', {
                    is: 'auth0',
                    then: yup.object({
                        issuer: yup.string().required(),
                        client_id: yup.string().required(),
                    }),
                })
                .when('type', {
                    is: 'oidc',
                    then: yup.object({
                        client_id: yup.string().required(),
                        issuer: yup.string().required(),
                        mode: yup.string().required(),
                        client_secret: yup.string().when('mode', {
                            is: (value) => value === 'auto' || value === 'post',
                            then: yup.string().required(),
                        }),
                    }),
                })
                .when('type', {
                    is: 'saml',
                    then: yup.object({
                        configurationType: yup.string().required(),
                        sp_issuer: yup.string().required(),
                        idp_metadata_url: yup.string().when('mode', {
                            is: (value) => value === 'auto' || value === 'post',
                            then: yup.string().required(),
                        }),
                    }),
                }),
        }),
    });
    const { dirty, handleChange, isValid, setFieldValue, values } = formik;

    function onChange(_value, event) {
        handleChange(event);
    }

    function onClickSubmit() {
        const transformedValues = transformValuesBeforeSaving(values);

        // Still submitting via Redux for MVP of Scoped Access feature
        dispatch(authActions.saveAuthProvider(transformedValues));
    }

    const hasAction = Boolean(action);
    const isViewing = !hasAction;
    const formTitle =
        action === 'create'
            ? getNewAuthProviderTitle(selectedAuthProvider.type)
            : selectedAuthProvider.name;

    return (
        <Form>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">{formTitle}</Title>
                    </ToolbarItem>
                    {isActionable && (
                        <ToolbarGroup
                            alignment={{ default: 'alignRight' }}
                            spaceItems={{ default: 'spaceItemsLg' }}
                        >
                            {hasAction ? (
                                <ToolbarGroup variant="button-group">
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            onClick={onClickSubmit}
                                            isDisabled={!dirty || !isValid}
                                            isSmall
                                        >
                                            Save
                                        </Button>
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <Button variant="tertiary" onClick={onClickCancel} isSmall>
                                            Cancel
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            ) : (
                                <ToolbarItem>
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
                        </ToolbarGroup>
                    )}
                </ToolbarContent>
            </Toolbar>
            {!!saveAuthProviderError && (
                <Alert isInline variant="danger" title="Problem saving auth provider">
                    <p>{saveAuthProviderError?.message}</p>
                </Alert>
            )}
            <FormikProvider value={formik}>
                <FormSection title="Configuration" titleElement="h3" className="pf-u-mt-0">
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
                                    handleSelect={setFieldValue}
                                    isDisabled
                                >
                                    {availableAuthProviders.map(({ value, label }) => (
                                        <SelectOption key={value} value={value}>
                                            {label}
                                        </SelectOption>
                                    ))}
                                </SelectSingle>
                            </FormGroup>
                        </GridItem>
                        <ConfigurationFormFields
                            config={values.config}
                            isViewing={isViewing}
                            onChange={onChange}
                            setFieldValue={setFieldValue}
                            type={values.type}
                        />
                    </Grid>
                </FormSection>
                <FormSection
                    title={`Assign roles to your ${selectedAuthProvider.type} users`}
                    titleElement="h3"
                >
                    <FormGroup
                        className="pf-u-w-100 pf-u-w-75-on-md pf-u-w-50-on-lg"
                        label="Minimum access role"
                        fieldId="minimumAccessRole"
                        isRequired
                    >
                        <SelectSingle
                            id="defaultRole"
                            value={values.defaultRole} // TODO see getDefaultRoleByAuthProviderId in classic code
                            handleSelect={setFieldValue}
                            isDisabled={isViewing}
                        >
                            {roles.map(({ name }) => (
                                <SelectOption key={name} value={name} />
                            ))}
                        </SelectSingle>
                    </FormGroup>
                    <div id="minimum-access-role-description">
                        <Alert isInline variant="info" title="">
                            <p>
                                The minimum access role is granted to all users who sign in with
                                this authentication provider.
                            </p>
                            <p>
                                To give users different roles, add rules. Users are granted all
                                matching roles.
                            </p>
                            <p>
                                Set the minimum access role to <strong>None</strong> if you want to
                                define permissions completely using specific rules below.
                            </p>
                        </Alert>
                    </div>
                    <FormSection title="Rules" titleElement="h3" className="pf-u-mt-0">
                        <RuleGroups
                            groups={values.groups}
                            roles={roles}
                            onChange={onChange}
                            setFieldValue={setFieldValue}
                        />
                    </FormSection>
                </FormSection>
            </FormikProvider>
        </Form>
    );
}

export default AuthProviderForm;

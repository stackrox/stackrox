/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { useHistory, Link } from 'react-router-dom';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useFormik, FormikProvider } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    Button,
    Flex,
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
    ValidatedOptions,
} from '@patternfly/react-core';

import SelectSingle from 'Components/SelectSingle'; // TODO import from where?
import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';

import { AuthProvider, getIsAuthProviderImmutable } from 'services/AuthService';
import ConfigurationFormFields from './ConfigurationFormFields';
import RuleGroups, { RuleGroupErrors } from './RuleGroups';
import {
    getInitialAuthProviderValues,
    transformInitialValues,
    transformValuesBeforeSaving,
    getGroupsByAuthProviderId,
    getDefaultRoleByAuthProviderId,
    isDefaultGroupModifiable,
} from './authProviders.utils';
import { AccessControlQueryAction } from '../accessControlPaths';
import { TraitsOriginLabel } from '../TraitsOriginLabel';

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
    saveAuthProviderStatus: selectors.getSaveAuthProviderStatus,
    availableProviderTypes: selectors.getAvailableProviderTypes,
});

function getNewAuthProviderTitle(type, availableProviderTypes) {
    const selectedType = availableProviderTypes.find(({ value }) => value === type);

    return `Create ${selectedType?.label as string} provider`;
}

function getRuleAttributes(type, availableProviderTypes) {
    return (
        (availableProviderTypes.find(({ value }) => value === type)?.ruleAttributes as string[]) ||
        []
    );
}

function testModeSupported(provider) {
    return (
        provider.type === 'auth0' ||
        provider.type === 'oidc' ||
        provider.type === 'saml' ||
        provider.type === 'openshift'
    );
}

function AuthProviderForm({
    isActionable,
    action,
    selectedAuthProvider,
    onClickCancel,
    onClickEdit,
}: AuthProviderFormProps): ReactElement {
    const history = useHistory();
    const { groups, roles, saveAuthProviderStatus, availableProviderTypes } =
        useSelector(authProviderState);
    const dispatch = useDispatch();

    const initialValues = !selectedAuthProvider.name
        ? getInitialAuthProviderValues(selectedAuthProvider)
        : { ...selectedAuthProvider };
    const filteredGroups = getGroupsByAuthProviderId(groups, selectedAuthProvider.id);
    const defaultRole = getDefaultRoleByAuthProviderId(groups, selectedAuthProvider.id);
    const canChangeDefaultRole = isDefaultGroupModifiable(groups, selectedAuthProvider.id);

    const modifiedInitialValues = {
        ...transformInitialValues(initialValues),
        groups: filteredGroups,
        defaultRole,
    };

    const authProviderValidationSchema = yup.object().shape({
        name: yup.string().required('A name is required.'),
        type: yup.string().required(),
        defaultRole: yup.string().required(),
        groups: yup.array().of(
            yup.object().shape({
                roleName: yup.string().required('Role is a required field'),
                props: yup.object().shape({
                    key: yup.string().required('Key is a required field'),
                    value: yup.string().required('Value is a required field'),
                }),
            })
        ),
        /* eslint-disable @typescript-eslint/no-unsafe-return */
        config: yup
            .object()
            .when('type', {
                is: 'auth0',
                then: (configSchema) =>
                    configSchema.shape({
                        issuer: yup.string().required('An issuer is required.'),
                        client_id: yup.string().required('A client ID is required.'),
                    }),
            })
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            .when('type', {
                is: 'oidc',
                then: (configSchema) =>
                    configSchema.shape({
                        client_id: yup.string().required('A client ID is required.'),
                        issuer: yup.string().required('An issuer is required.'),
                        mode: yup.string().required(), // selected from a list where one is always selected
                        client_secret: yup
                            .string()
                            .when(['mode', 'do_not_use_client_secret', 'clientOnly'], {
                                is: (mode, do_not_use_client_secret, clientOnly) =>
                                    (mode === 'auto' || mode === 'post' || mode === 'query') &&
                                    !do_not_use_client_secret &&
                                    !clientOnly?.clientSecretStored,
                                then: (clientSecretSchema) =>
                                    clientSecretSchema.required('A client secret is required.'),
                            }),
                    }),
            })
            .when('type', {
                is: 'saml',
                then: (configSchema) =>
                    configSchema.shape({
                        configurationType: yup.string().required(), // selected from a list where one is always selected
                        sp_issuer: yup.string().required('A service provider issuer is required.'),
                        idp_metadata_url: yup.string().when('configurationType', {
                            is: (value) => value === 'dynamic',
                            then: (schema) =>
                                schema
                                    .required('An IdP metadata URL is required.')
                                    .url(
                                        'Must be a valid URL, for example, https://idp.example.com/metadata'
                                    ),
                        }),
                        idp_issuer: yup.string().when('configurationType', {
                            is: (value) => value === 'static',
                            then: (schema) => schema.required('An IdP issuer is required.'),
                        }),
                        idp_sso_url: yup.string().when('configurationType', {
                            is: (value) => value === 'static',
                            then: (schema) =>
                                schema
                                    .required('An IdP SSO URL is required.')
                                    .url(
                                        'Must be a valid URL, for example, https://idp.example.com/login'
                                    ),
                        }),
                        idp_cert_pem: yup.string().when('configurationType', {
                            is: (value) => value === 'static',
                            then: (schema) =>
                                schema.required('One or more IdP certificate (PEM) is required.'),
                        }),
                    }),
            })
            .when('type', {
                is: 'userpki',
                then: (configSchema) =>
                    configSchema.shape({
                        keys: yup
                            .string()
                            .required('One or more CA certificates (PEM) is required.'),
                    }),
            })
            .when('type', {
                is: 'iap',
                then: (configSchema) =>
                    configSchema.shape({
                        audience: yup.string().required('An audience is required.'),
                    }),
            }),
        /* eslint-enable @typescript-eslint/no-unsafe-return */
    });

    const formik = useFormik({
        initialValues: modifiedInitialValues,
        onSubmit: () => {},
        validationSchema: authProviderValidationSchema,
        enableReinitialize: true,
    });
    const { dirty, handleChange, isValid, setFieldValue, handleBlur, values, errors, touched } =
        formik;

    function onChange(_value, event) {
        handleChange(event);
    }

    function handleTest() {
        const windowFeatures =
            'location=no,menubar=no,scrollbars=yes,toolbar=no,width=768,height=512,left=0,top=0'; // browser not required to honor these attrs

        const windowObjectReference = window.open(
            `/sso/login/${selectedAuthProvider.id}?test=true`,
            `Test Login for ${selectedAuthProvider.name}`,
            windowFeatures
        );

        if (windowObjectReference) {
            windowObjectReference.focus();
        }
    }

    function onClickSubmit() {
        dispatch(authActions.setSaveAuthProviderStatus(null));

        const transformedValues = transformValuesBeforeSaving(values);

        // Still submitting via Redux for MVP of Scoped Access feature
        dispatch(authActions.saveAuthProvider(transformedValues));
    }

    // handle relevant saving statuses
    if (saveAuthProviderStatus?.status === 'success') {
        dispatch(authActions.setSaveAuthProviderStatus(null));

        // Go back from action=create to list.
        history.goBack();
    }
    const isSaving = saveAuthProviderStatus?.status === 'saving';

    const hasAction = Boolean(action);
    const isViewing = !hasAction;
    const formTitle =
        action === 'create'
            ? getNewAuthProviderTitle(selectedAuthProvider.type, availableProviderTypes)
            : selectedAuthProvider.name;

    const ruleAttributes = getRuleAttributes(selectedAuthProvider.type, availableProviderTypes);

    return (
        <Form>
            <Toolbar inset={{ default: 'insetNone' }} className="pf-u-pt-0">
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2">{formTitle}</Title>
                    </ToolbarItem>
                    {action !== 'create' && (
                        <ToolbarItem>
                            <TraitsOriginLabel traits={selectedAuthProvider.traits} />
                        </ToolbarItem>
                    )}
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
                                            isLoading={isSaving}
                                            spinnerAriaValueText={isSaving ? 'Saving' : undefined}
                                        >
                                            {isSaving ? 'Saving...' : 'Save'}
                                        </Button>
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <Button variant="tertiary" onClick={onClickCancel} isSmall>
                                            Cancel
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            ) : (
                                <ToolbarGroup variant="button-group">
                                    <ToolbarItem>
                                        <Button variant="link" isSmall>
                                            <Link
                                                to="/main/access-control/auth-providers"
                                                aria-current="page"
                                            >
                                                Return to auth providers list
                                            </Link>
                                        </Button>
                                    </ToolbarItem>
                                    {testModeSupported(selectedAuthProvider) &&
                                        selectedAuthProvider.id && (
                                            <ToolbarItem>
                                                <Button
                                                    variant="secondary"
                                                    onClick={handleTest}
                                                    isDisabled={action === 'edit'}
                                                    isSmall
                                                >
                                                    Test login
                                                </Button>
                                            </ToolbarItem>
                                        )}
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            onClick={onClickEdit}
                                            isDisabled={action === 'edit'}
                                            isSmall
                                        >
                                            {selectedAuthProvider.active ||
                                            getIsAuthProviderImmutable(selectedAuthProvider)
                                                ? 'Edit minimum role and rules'
                                                : 'Edit auth provider'}
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            )}
                        </ToolbarGroup>
                    )}
                </ToolbarContent>
            </Toolbar>
            {saveAuthProviderStatus?.status === 'error' && (
                <Alert isInline variant="danger" title="Problem saving auth provider">
                    <p>{saveAuthProviderStatus?.message}</p>
                </Alert>
            )}
            {testModeSupported(selectedAuthProvider) &&
                selectedAuthProvider.id &&
                !selectedAuthProvider.active && (
                    <Alert
                        isInline
                        variant="info"
                        title={
                            <span>
                                Click <em>Test login</em> to check that your authentication provider
                                is working properly.
                            </span>
                        }
                    />
                )}
            {selectedAuthProvider.active && (
                <Alert
                    isInline
                    variant="warning"
                    title={
                        <span>
                            For auth providers that have been logged into, you can only edit the
                            minimum role and rules. If you need to change the configuration, please
                            delete and recreate.
                        </span>
                    }
                />
            )}
            {getIsAuthProviderImmutable(selectedAuthProvider) && (
                <Alert
                    isInline
                    variant="warning"
                    title={
                        <span>
                            This auth provider is immutable. You can only edit the minimum role and
                            rules.
                        </span>
                    }
                />
            )}
            <FormikProvider value={formik}>
                <FormSection title="Configuration" titleElement="h3" className="pf-u-mt-0">
                    <Grid hasGutter>
                        <GridItem span={12} lg={6}>
                            <FormGroup
                                label="Name"
                                fieldId="name"
                                isRequired
                                helperTextInvalid={errors.name || ''}
                                validated={
                                    errors.name && touched.name ? ValidatedOptions.error : 'default'
                                }
                            >
                                <TextInput
                                    type="text"
                                    id="name"
                                    value={values.name}
                                    onChange={onChange}
                                    isDisabled={
                                        isViewing ||
                                        values.active ||
                                        getIsAuthProviderImmutable(values)
                                    }
                                    isRequired
                                    onBlur={handleBlur}
                                    validated={
                                        errors.name && touched.name
                                            ? ValidatedOptions.error
                                            : 'default'
                                    }
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
                                    {availableProviderTypes.map(({ value, label }) => (
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
                            onBlur={handleBlur}
                            configErrors={errors.config}
                            configTouched={touched.config}
                            disabled={values.active || getIsAuthProviderImmutable(values)}
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
                            isDisabled={isViewing || !canChangeDefaultRole}
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
                    {selectedAuthProvider.requiredAttributes &&
                        selectedAuthProvider.requiredAttributes.length > 0 && (
                            <FormSection
                                title="Required attributes for the authentication provider"
                                titleElement="h3"
                            >
                                {selectedAuthProvider.requiredAttributes.map(
                                    (attribute, index: number) => (
                                        <Flex
                                            key={`${attribute.attributeKey}_required_attribute_${index}`}
                                        >
                                            <FormGroup label="Key" fieldId={attribute.attributeKey}>
                                                <TextInput
                                                    type="text"
                                                    id={attribute.attributeKey}
                                                    value={attribute.attributeKey}
                                                    isDisabled
                                                />
                                            </FormGroup>
                                            <FormGroup
                                                label="Value"
                                                fieldId={attribute.attributeValue}
                                            >
                                                <TextInput
                                                    type="text"
                                                    id={attribute.attributeValue}
                                                    value={attribute.attributeValue}
                                                    isDisabled
                                                />
                                            </FormGroup>
                                        </Flex>
                                    )
                                )}
                                <div id="required-attributes-description">
                                    <Alert isInline variant="info" title="">
                                        <p>
                                            The required attributes are used to require attributes
                                            being returned from the authentication provider.
                                        </p>
                                        <p>
                                            In case a required attribute is not set, the login will
                                            fail and no role will be set to the user.
                                        </p>
                                    </Alert>
                                </div>
                            </FormSection>
                        )}
                    <FormSection title="Rules" titleElement="h3" className="pf-u-mt-0">
                        <RuleGroups
                            authProviderId={selectedAuthProvider.id}
                            groups={values.groups}
                            roles={roles}
                            onChange={onChange}
                            setFieldValue={setFieldValue}
                            disabled={isViewing}
                            errors={errors?.groups as RuleGroupErrors[]}
                            ruleAttributes={ruleAttributes}
                        />
                    </FormSection>
                </FormSection>
            </FormikProvider>
        </Form>
    );
}

export default AuthProviderForm;

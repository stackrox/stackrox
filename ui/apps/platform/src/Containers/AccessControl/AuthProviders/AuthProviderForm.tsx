/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useSelector, useDispatch } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useFormik, FormikProvider, FieldArray } from 'formik';
import * as yup from 'yup';
import {
    Alert,
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    FormSection,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    TextInput,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    Tooltip,
    ValidatedOptions,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { InfoCircleIcon, PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';

import SelectSingle from 'Components/SelectSingle'; // TODO import from where?
import { selectors } from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { Role } from 'services/RolesService';
import {
    AuthProvider,
    AuthProviderInfo,
    getIsAuthProviderImmutable,
    Group,
} from 'services/AuthService';

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
import { isUserResource } from '../traits';

export type AuthProviderFormProps = {
    isActionable: boolean;
    action?: AccessControlQueryAction;
    selectedAuthProvider: AuthProvider;
    onClickCancel: () => void;
    onClickEdit: () => void;
};

type AuthProviderState = {
    roles: Role[];
    groups: Group[];
    saveAuthProviderStatus: { status: string; message: string } | null;
    availableProviderTypes: AuthProviderInfo[];
};

const authProviderState = createStructuredSelector<AuthProviderState>({
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
    const navigate = useNavigate();
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
        requiredAttributes: yup.array().of(
            yup.object().shape({
                attributeKey: yup.string().required('Key is a required field'),
                attributeValue: yup.string().required('Value is a required field'),
            })
        ),
        claimMappings: yup
            .array()
            .of(yup.array().of(yup.string().required('Empty value is not allowed')).length(2))
            .test('uniqueness test', 'Claim mappings should contain unique keys', (value) => {
                if (!value) {
                    return true;
                }
                const keys = value.map((mapping) => {
                    return mapping ? mapping[0] : '';
                });
                return keys.length === new Set(keys).size;
            }),
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
    });

    const formik = useFormik({
        initialValues: modifiedInitialValues,
        onSubmit: () => {},
        validationSchema: authProviderValidationSchema,
        enableReinitialize: true,
    });
    const { dirty, handleChange, isValid, setFieldValue, handleBlur, values, errors, touched } =
        formik;

    function onChange(event: React.FormEvent) {
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
        navigate(-1);
    }
    const isSaving = saveAuthProviderStatus?.status === 'saving';

    const hasAction = Boolean(action);
    const isViewing = !hasAction;
    const formTitle =
        action === 'create'
            ? getNewAuthProviderTitle(selectedAuthProvider.type, availableProviderTypes)
            : selectedAuthProvider.name;

    const ruleAttributes = getRuleAttributes(selectedAuthProvider.type, availableProviderTypes);

    const isDisabled = isViewing || values.active || getIsAuthProviderImmutable(values);
    const nameValidated = errors.name && touched.name ? ValidatedOptions.error : 'default';

    return (
        <Form>
            <Toolbar inset={{ default: 'insetNone' }} className="pf-v5-u-pt-0">
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h1">{formTitle}</Title>
                    </ToolbarItem>
                    {action !== 'create' && (
                        <ToolbarItem>
                            <TraitsOriginLabel traits={selectedAuthProvider.traits} />
                        </ToolbarItem>
                    )}
                    {isActionable && (
                        <ToolbarGroup
                            align={{ default: 'alignRight' }}
                            spaceItems={{ default: 'spaceItemsLg' }}
                        >
                            {hasAction ? (
                                <ToolbarGroup variant="button-group">
                                    <ToolbarItem>
                                        <Button
                                            variant="primary"
                                            onClick={onClickSubmit}
                                            isDisabled={!dirty || !isValid}
                                            size="sm"
                                            isLoading={isSaving}
                                            spinnerAriaValueText={isSaving ? 'Saving' : undefined}
                                        >
                                            {isSaving ? 'Saving...' : 'Save'}
                                        </Button>
                                    </ToolbarItem>
                                    <ToolbarItem>
                                        <Button
                                            variant="tertiary"
                                            onClick={onClickCancel}
                                            size="sm"
                                        >
                                            Cancel
                                        </Button>
                                    </ToolbarItem>
                                </ToolbarGroup>
                            ) : (
                                <ToolbarGroup variant="button-group">
                                    <ToolbarItem>
                                        <Link
                                            to="/main/access-control/auth-providers"
                                            aria-current="page"
                                            className="pf-v5-u-font-size-sm"
                                        >
                                            Return to auth providers list
                                        </Link>
                                    </ToolbarItem>
                                    {testModeSupported(selectedAuthProvider) &&
                                        selectedAuthProvider.id && (
                                            <ToolbarItem>
                                                <Button
                                                    variant="secondary"
                                                    onClick={handleTest}
                                                    isDisabled={action === 'edit'}
                                                    size="sm"
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
                                            size="sm"
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
                <Alert isInline variant="danger" title="Problem saving auth provider" component="p">
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
                        component="p"
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
                    component="p"
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
                    component="p"
                />
            )}
            <FormikProvider value={formik}>
                <FormSection title="Configuration" titleElement="h2" className="pf-v5-u-mt-0">
                    <Grid hasGutter>
                        <GridItem span={12} lg={6}>
                            <FormGroup label="Name" fieldId="name" isRequired>
                                <TextInput
                                    type="text"
                                    id="name"
                                    value={values.name}
                                    onChange={onChange}
                                    isDisabled={isDisabled}
                                    isRequired
                                    onBlur={handleBlur}
                                    validated={
                                        errors.name && touched.name
                                            ? ValidatedOptions.error
                                            : 'default'
                                    }
                                />
                                <FormHelperText>
                                    <HelperText>
                                        <HelperTextItem variant={nameValidated}>
                                            {errors.name ? errors.name : ''}
                                        </HelperTextItem>
                                    </HelperText>
                                </FormHelperText>
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
                            isAuthProviderActive={values.active}
                            isAuthProviderDeclarative={getIsAuthProviderImmutable(values)}
                        />
                    </Grid>
                </FormSection>
                <FormSection
                    title={`Assign roles to your ${selectedAuthProvider.type} users`}
                    titleElement="h2"
                >
                    <FormGroup
                        className="pf-v5-u-w-100 pf-v5-u-w-75-on-md pf-v5-u-w-50-on-lg"
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
                        <Alert
                            isInline
                            variant="info"
                            title="Note: the minimum access role is granted to all users who sign in with
                                this authentication provider."
                            component="p"
                        >
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
                    {selectedAuthProvider.type === 'oidc' && (
                        <FormSection
                            title="Required attributes for the authentication provider"
                            titleElement="h2"
                        >
                            <Alert
                                isInline
                                variant="info"
                                title="Note: the required attributes are used to require attributes being
                                    returned from the authentication provider."
                                component="p"
                            >
                                <p>
                                    In case a required attribute is not returned from the
                                    authentication provider, the login attempt will fail and no role
                                    will be assigned to the user.
                                </p>
                            </Alert>
                            {(!values.requiredAttributes ||
                                values.requiredAttributes.length === 0) && (
                                <p>No required attributes defined</p>
                            )}
                            <FieldArray
                                name="requiredAttributes"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.requiredAttributes &&
                                            values.requiredAttributes.map(
                                                (attribute, index: number) => (
                                                    <Flex key={`required_attribute_${index}`}>
                                                        <FormGroup
                                                            label="Key"
                                                            fieldId={`requiredAttributes[${index}].attributeKey`}
                                                        >
                                                            <TextInput
                                                                type="text"
                                                                id={`requiredAttributes[${index}].attributeKey`}
                                                                value={attribute.attributeKey}
                                                                onChange={onChange}
                                                                isDisabled={isDisabled}
                                                            />
                                                        </FormGroup>
                                                        <FormGroup
                                                            label="Value"
                                                            fieldId={`requiredAttributes[${index}].attributeValue`}
                                                        >
                                                            <TextInput
                                                                type="text"
                                                                id={`requiredAttributes[${index}].attributeValue`}
                                                                value={attribute.attributeValue}
                                                                onChange={onChange}
                                                                isDisabled={isDisabled}
                                                            />
                                                        </FormGroup>
                                                        {!isDisabled && (
                                                            <FlexItem>
                                                                <Button
                                                                    variant="plain"
                                                                    aria-label="Delete required attribute"
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
                                                        {!isUserResource(
                                                            selectedAuthProvider.traits
                                                        ) && (
                                                            <FlexItem>
                                                                <Tooltip content="Auth provider is managed declaratively and can only be edited declaratively.">
                                                                    <Button
                                                                        variant="plain"
                                                                        aria-label="Information button"
                                                                        style={{
                                                                            transform:
                                                                                'translate(0, 42px)',
                                                                        }}
                                                                    >
                                                                        <InfoCircleIcon />
                                                                    </Button>
                                                                </Tooltip>
                                                            </FlexItem>
                                                        )}
                                                    </Flex>
                                                )
                                            )}
                                        {!isDisabled && (
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
                                                                attributeKey: '',
                                                                attributeValue: '',
                                                            })
                                                        }
                                                    >
                                                        Add required attribute
                                                    </Button>
                                                </FlexItem>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </FormSection>
                    )}
                    {selectedAuthProvider.type === 'oidc' && (
                        <FormSection
                            title="Claim mappings for the authentication provider"
                            titleElement="h2"
                        >
                            <Alert
                                isInline
                                variant="info"
                                title="Note: the claim mappings are used to map claims returned in underlying identity provider’s token to ACS token."
                                component="p"
                            >
                                <p>
                                    In case claim mapping is not configured for a certain claim,
                                    this claim will not be returned from authentication provider.
                                </p>
                            </Alert>
                            {(!Array.isArray(values.claimMappings) ||
                                values.claimMappings.length === 0) && (
                                <p>No claim mappings defined</p>
                            )}
                            <FieldArray
                                name="claimMappings"
                                render={(arrayHelpers) => (
                                    <>
                                        {Array.isArray(values.claimMappings) &&
                                            values.claimMappings.map((mapping, index: number) => (
                                                <Flex key={`claim_mapping_${index}`}>
                                                    <FormGroup
                                                        label="Key"
                                                        fieldId={`claimMappings[${index}][0]`}
                                                    >
                                                        <TextInput
                                                            type="text"
                                                            id={`claimMappings[${index}][0]`}
                                                            value={mapping[0]}
                                                            onChange={onChange}
                                                            isDisabled={isDisabled}
                                                        />
                                                    </FormGroup>
                                                    <FormGroup
                                                        label="Value"
                                                        fieldId={`claimMappings[${index}][1]`}
                                                    >
                                                        <TextInput
                                                            type="text"
                                                            id={`claimMappings[${index}][1]`}
                                                            value={mapping[1]}
                                                            onChange={onChange}
                                                            isDisabled={isDisabled}
                                                        />
                                                    </FormGroup>
                                                    {!isDisabled && (
                                                        <FlexItem>
                                                            <Button
                                                                variant="plain"
                                                                aria-label="Delete claim mapping"
                                                                style={{
                                                                    transform: 'translate(0, 42px)',
                                                                }}
                                                                onClick={() =>
                                                                    arrayHelpers.remove(index)
                                                                }
                                                            >
                                                                <TrashIcon />
                                                            </Button>
                                                        </FlexItem>
                                                    )}
                                                    {!isUserResource(
                                                        selectedAuthProvider.traits
                                                    ) && (
                                                        <FlexItem>
                                                            <Tooltip content="Auth provider is managed declaratively and can only be edited declaratively.">
                                                                <Button
                                                                    variant="plain"
                                                                    aria-label="Information button"
                                                                    style={{
                                                                        transform:
                                                                            'translate(0, 42px)',
                                                                    }}
                                                                >
                                                                    <InfoCircleIcon />
                                                                </Button>
                                                            </Tooltip>
                                                        </FlexItem>
                                                    )}
                                                </Flex>
                                            ))}
                                        {!isDisabled && (
                                            <Flex>
                                                <FlexItem>
                                                    <Button
                                                        variant="link"
                                                        isInline
                                                        icon={
                                                            <PlusCircleIcon className="pf-v5-u-mr-sm" />
                                                        }
                                                        onClick={() => arrayHelpers.push(['', ''])}
                                                    >
                                                        Add claim mapping
                                                    </Button>
                                                </FlexItem>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </FormSection>
                    )}
                    <FormSection title="Rules" titleElement="h2" className="pf-v5-u-mt-0">
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

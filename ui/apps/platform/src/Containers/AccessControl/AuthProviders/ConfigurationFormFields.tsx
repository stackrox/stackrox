import React, { ReactElement } from 'react';
import { FormikErrors, FormikTouched } from 'formik';
import {
    Alert,
    Checkbox,
    FormGroup,
    GridItem,
    SelectOption,
    TextArea,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';

import { oidcCallbackModes } from 'constants/accessControl';
import { AuthProviderConfig, AuthProviderType } from 'services/AuthService';
import SelectSingle from 'Components/SelectSingle'; // TODO import from where?

export type ConfigurationFormFieldsProps = {
    config: AuthProviderConfig;
    isViewing: boolean;
    onChange: (
        _value: unknown,
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) => void;
    onBlur: (
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) => void;
    setFieldValue: (name: string, value: string | boolean) => void;
    type: AuthProviderType;
    configErrors?: FormikErrors<Record<string, string>>;
    configTouched?: FormikTouched<Record<string, string>>;
    isAuthProviderActive: boolean | undefined;
    isAuthProviderDeclarative: boolean;
};

const baseURL = `${window.location.protocol}//${window.location.host}`;
const oidcFragmentCallbackURL = `${baseURL}/auth/response/oidc`;
const oidcPostCallbackURL = `${baseURL}/sso/providers/oidc/callback`;
const samlACSURL = `${baseURL}/sso/providers/saml/acs`;

function getClientSecretHelperText(config, clientSecretSupported) {
    // use client secret placeholder as an explanation text
    let clientSecretHelperText = 'Client Secret provided by your IdP';
    if (!clientSecretSupported) {
        clientSecretHelperText = 'Client Secret is not supported with Fragment callback mode';
    } else if (config?.clientOnly?.clientSecretStored) {
        clientSecretHelperText = config?.do_not_use_client_secret
            ? 'Disabled, the currently stored secret will be removed'
            : 'Leave this field empty to keep the currently stored secret';
    } else if (config?.do_not_use_client_secret) {
        clientSecretHelperText = 'Disabled';
    }

    return clientSecretHelperText;
}
function ConfigurationFormFields({
    isViewing,
    onChange,
    onBlur,
    config,
    setFieldValue,
    type,
    configErrors,
    configTouched,
    isAuthProviderActive = false,
    isAuthProviderDeclarative = false,
}: ConfigurationFormFieldsProps): ReactElement {
    const clientSecretSupported = config.mode !== 'fragment';
    const clientSecretMandatory = config.mode === 'query';

    const clientSecretHelperText = getClientSecretHelperText(config, clientSecretSupported);
    const isActiveModificationsDisabled = isAuthProviderActive || isAuthProviderDeclarative;
    const doNotUseClientSecretDisabled =
        isActiveModificationsDisabled || !clientSecretSupported || clientSecretMandatory;

    const showIssuerError = Boolean(configErrors?.issuer && configTouched?.issuer);
    const showClientIdError = Boolean(configErrors?.client_id && configTouched?.client_id);
    const showClientSecretError = Boolean(
        configErrors?.client_secret && configTouched?.client_secret
    );
    const showSpIssuerError = Boolean(configErrors?.sp_issuer && configTouched?.sp_issuer);
    const showIdpMetadataUrlError = Boolean(
        configErrors?.idp_metadata_url && configTouched?.idp_metadata_url
    );
    const showIdpIssuerError = Boolean(configErrors?.idp_issuer && configTouched?.idp_issuer);
    const showIdpSsoUrlError = Boolean(configErrors?.idp_sso_url && configTouched?.idp_sso_url);
    const showIdpCertPemError = Boolean(configErrors?.idp_cert_pem && configTouched?.idp_cert_pem);
    const showUserPkiKeysError = Boolean(configErrors?.keys && configTouched?.keys);
    const showAudienceError = Boolean(configErrors?.audience && configTouched?.audience);

    function updateClientSecretFlagOnChange(name: string, value: string) {
        setFieldValue(name, value);
        if (value === 'fragment' && config.do_not_use_client_secret !== true) {
            setFieldValue('config.do_not_use_client_secret', true);
        } else if (value !== 'fragment' && config.do_not_use_client_secret !== false) {
            setFieldValue('config.do_not_use_client_secret', false);
        }
    }

    const clientOnly = config.clientOnly as Record<string, boolean>;

    return (
        <>
            {type === 'auth0' && (
                <>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Auth0 tenant"
                            fieldId="config.issuer"
                            isRequired
                            helperText={
                                <span className="pf-u-font-size-sm">
                                    for example,{' '}
                                    <kbd className="pf-u-font-size-xs">your-tenant.auth0.com</kbd>
                                </span>
                            }
                            helperTextInvalid={configErrors?.issuer || ''}
                            validated={showIssuerError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="text"
                                id="config.issuer"
                                value={(config.issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                                isRequired
                                onBlur={onBlur}
                                validated={showIssuerError ? ValidatedOptions.error : 'default'}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Client ID"
                            fieldId="config.client_id"
                            isRequired
                            helperTextInvalid={configErrors?.client_id || ''}
                            validated={showClientIdError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="text"
                                id="config.client_id"
                                value={(config.client_id as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                                isRequired
                                onBlur={onBlur}
                                validated={showClientIdError ? ValidatedOptions.error : 'default'}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12}>
                        <Alert
                            isInline
                            variant="info"
                            title={
                                <span>
                                    Note: if required by your IdP, use the following callback URL:
                                </span>
                            }
                        >
                            <p>{oidcFragmentCallbackURL}</p>
                        </Alert>
                    </GridItem>
                </>
            )}
            {type === 'oidc' && (
                <>
                    <GridItem span={12} lg={6}>
                        <FormGroup label="Callback mode" fieldId="config.mode" isRequired>
                            <SelectSingle
                                id="config.mode"
                                value={config.mode as string}
                                handleSelect={updateClientSecretFlagOnChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                            >
                                {oidcCallbackModes.map(({ value, label }) => (
                                    <SelectOption key={value} value={value}>
                                        {label}
                                    </SelectOption>
                                ))}
                            </SelectSingle>
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Issuer"
                            fieldId="config.issuer"
                            isRequired
                            helperText={
                                <span className="pf-u-font-size-sm">
                                    for example,{' '}
                                    <kbd className="pf-u-font-size-xs">
                                        tenant.auth-provider.com
                                    </kbd>
                                </span>
                            }
                            helperTextInvalid={configErrors?.issuer || ''}
                            validated={showIssuerError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="text"
                                id="config.issuer"
                                value={(config.issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                                isRequired
                                onBlur={onBlur}
                                validated={showIssuerError ? ValidatedOptions.error : 'default'}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Client ID"
                            fieldId="config.client_id"
                            isRequired
                            helperTextInvalid={configErrors?.client_id || ''}
                            validated={showClientIdError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="text"
                                id="config.client_id"
                                value={(config.client_id as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                                isRequired
                                onBlur={onBlur}
                                validated={showClientIdError ? ValidatedOptions.error : 'default'}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Client Secret"
                            fieldId="config.client_secret"
                            isRequired={
                                !(
                                    config.mode === 'fragment' ||
                                    config.do_not_use_client_secret ||
                                    clientOnly?.clientSecretStored
                                )
                            }
                            helperText={
                                <span className="pf-u-font-size-sm">{clientSecretHelperText}</span>
                            }
                            helperTextInvalid={configErrors?.client_secret || ''}
                            validated={showClientSecretError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="password"
                                id="config.client_secret"
                                value={(config.client_secret as string) || ''}
                                onChange={onChange}
                                isDisabled={
                                    isViewing ||
                                    isActiveModificationsDisabled ||
                                    config.mode === 'fragment' ||
                                    !!config.do_not_use_client_secret
                                }
                                isRequired={
                                    !(config.mode === 'fragment' || config.do_not_use_client_secret)
                                }
                                onBlur={onBlur}
                                validated={
                                    showClientSecretError ? ValidatedOptions.error : 'default'
                                }
                                placeholder={isViewing ? '*****' : ''}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={6} lg={6}>
                        <FormGroup fieldId="config.do_not_use_client_secret">
                            <Checkbox
                                isChecked={
                                    config.mode !== 'query'
                                        ? (config.do_not_use_client_secret as boolean)
                                        : false
                                }
                                label="Do not use Client Secret (not recommended)"
                                id="config.do_not_use_client_secret"
                                name="config.do_not_use_client_secret"
                                aria-label="Do not use Client Secret (not recommended)"
                                onChange={onChange}
                                isDisabled={isViewing || doNotUseClientSecretDisabled}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={6} lg={6}>
                        <FormGroup
                            fieldId="config.disable_offline_access_scope"
                            helperText={
                                <span className="pf-u-font-size-sm">
                                    Use if the identity provider has a limit on the number of
                                    offline tokens that it can issue.
                                </span>
                            }
                        >
                            <Checkbox
                                isChecked={config.disable_offline_access_scope as boolean}
                                label="Disable 'offline_access' scope"
                                id="config.disable_offline_access_scope"
                                name="config.disable_offline_access_scope"
                                aria-label="Disable 'offline_access' scope"
                                onChange={(checked) => {
                                    setFieldValue('config.disable_offline_access_scope', checked);
                                }}
                                isDisabled={isViewing || isAuthProviderDeclarative}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12}>
                        <Alert
                            isInline
                            variant="info"
                            title={<span>Note: allow the following callback URLs:</span>}
                        >
                            <p>
                                {oidcFragmentCallbackURL}
                                <br />
                                {oidcPostCallbackURL}
                            </p>
                        </Alert>
                    </GridItem>
                </>
            )}
            {type === 'saml' && (
                <>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Service Provider issuer"
                            fieldId="config.sp_issuer"
                            isRequired
                            helperText={
                                <span className="pf-u-font-size-sm">
                                    for example,{' '}
                                    <kbd className="pf-u-font-size-xs">
                                        https://prevent.stackrox.io
                                    </kbd>
                                </span>
                            }
                            helperTextInvalid={configErrors?.sp_issuer || ''}
                            validated={showSpIssuerError ? ValidatedOptions.error : 'default'}
                        >
                            <TextInput
                                type="text"
                                id="config.sp_issuer"
                                value={(config.sp_issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                                isRequired
                                onBlur={onBlur}
                                validated={showSpIssuerError ? ValidatedOptions.error : 'default'}
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Configuration"
                            fieldId="config.configurationType"
                            isRequired
                        >
                            <SelectSingle
                                id="config.configurationType"
                                value={config.configurationType as string}
                                handleSelect={setFieldValue}
                                isDisabled={isViewing || isActiveModificationsDisabled}
                            >
                                <SelectOption value="dynamic">
                                    Option 1: Dynamic configuration
                                </SelectOption>
                                <SelectOption value="static">
                                    Option 2: Static configuration
                                </SelectOption>
                            </SelectSingle>
                        </FormGroup>
                    </GridItem>
                    {config.configurationType === 'dynamic' && (
                        <>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="IdP Metadata URL"
                                    fieldId="config.idp_metadata_url"
                                    isRequired
                                    helperText={
                                        <span className="pf-u-font-size-sm">
                                            for example,{' '}
                                            <kbd className="pf-u-font-size-xs">
                                                https://idp.example.com/metadata
                                            </kbd>
                                        </span>
                                    }
                                    helperTextInvalid={configErrors?.idp_metadata_url || ''}
                                    validated={
                                        showIdpMetadataUrlError ? ValidatedOptions.error : 'default'
                                    }
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_metadata_url"
                                        value={(config.idp_metadata_url as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing || isActiveModificationsDisabled}
                                        isRequired
                                        onBlur={onBlur}
                                        validated={
                                            showIdpMetadataUrlError
                                                ? ValidatedOptions.error
                                                : 'default'
                                        }
                                    />
                                </FormGroup>
                            </GridItem>
                        </>
                    )}
                    {config.configurationType === 'static' && (
                        <>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="IdP Issuer"
                                    fieldId="config.idp_issuer"
                                    isRequired={config.configurationType === 'static'}
                                    helperText={
                                        <span className="pf-u-font-size-sm">
                                            for example,{' '}
                                            <kbd className="pf-u-font-size-xs">
                                                https://idp.example.com/
                                            </kbd>
                                            {', '}
                                            or{' '}
                                            <kbd className="pf-u-font-size-xs">
                                                urn:something:else
                                            </kbd>
                                        </span>
                                    }
                                    helperTextInvalid={configErrors?.idp_issuer || ''}
                                    validated={
                                        showIdpIssuerError ? ValidatedOptions.error : 'default'
                                    }
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_issuer"
                                        value={(config.idp_issuer as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing || isActiveModificationsDisabled}
                                        isRequired={config.configurationType === 'static'}
                                        onBlur={onBlur}
                                        validated={
                                            showIdpIssuerError ? ValidatedOptions.error : 'default'
                                        }
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="IdP SSO URL"
                                    fieldId="config.idp_sso_url"
                                    isRequired={config.configurationType === 'static'}
                                    helperText={
                                        <span className="pf-u-font-size-sm">
                                            for example,{' '}
                                            <kbd className="pf-u-font-size-xs">
                                                https://idp.example.com/login
                                            </kbd>
                                        </span>
                                    }
                                    helperTextInvalid={configErrors?.idp_sso_url || ''}
                                    validated={
                                        showIdpSsoUrlError ? ValidatedOptions.error : 'default'
                                    }
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_sso_url"
                                        value={(config.idp_sso_url as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing || isActiveModificationsDisabled}
                                        isRequired={config.configurationType === 'static'}
                                        onBlur={onBlur}
                                        validated={
                                            showIdpSsoUrlError ? ValidatedOptions.error : 'default'
                                        }
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="Name/ID Format"
                                    fieldId="config.idp_nameid_format"
                                    helperText={
                                        <span className="pf-u-font-size-sm">
                                            for example,{' '}
                                            <kbd className="pf-u-font-size-xs">
                                                urn:oasis:names:tc:SAML:1.1:nameid-format:persistent
                                            </kbd>
                                        </span>
                                    }
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_nameid_format"
                                        value={(config.idp_nameid_format as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing || isActiveModificationsDisabled}
                                        onBlur={onBlur}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="IdP Certificate(s) (PEM)"
                                    fieldId="config.idp_cert_pem"
                                    isRequired={config.configurationType === 'static'}
                                    helperTextInvalid={configErrors?.idp_cert_pem || ''}
                                    validated={
                                        showIdpCertPemError ? ValidatedOptions.error : 'default'
                                    }
                                >
                                    <TextArea
                                        className="certificate-input"
                                        autoResize
                                        resizeOrientation="vertical"
                                        id="config.idp_cert_pem"
                                        value={(config.idp_cert_pem as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing || isActiveModificationsDisabled}
                                        isRequired={config.configurationType === 'static'}
                                        placeholder={
                                            '-----BEGIN CERTIFICATE-----\nYour certificate data\n-----END CERTIFICATE-----'
                                        }
                                        onBlur={onBlur}
                                        validated={
                                            showIdpCertPemError ? ValidatedOptions.error : 'default'
                                        }
                                    />
                                </FormGroup>
                            </GridItem>
                        </>
                    )}
                    <GridItem span={12}>
                        <Alert
                            isInline
                            variant="info"
                            title={
                                <span>
                                    Note: if required by your IdP, use the following Assertion
                                    Consumer Service (ACS) URL:
                                </span>
                            }
                        >
                            <p>{samlACSURL}</p>
                        </Alert>
                    </GridItem>
                </>
            )}
            {type === 'userpki' && (
                <GridItem span={12} lg={6}>
                    <FormGroup
                        label="CA certificate(s) (PEM)"
                        fieldId="config.keys"
                        isRequired
                        helperTextInvalid={configErrors?.keys || ''}
                        validated={showUserPkiKeysError ? ValidatedOptions.error : 'default'}
                    >
                        <TextArea
                            className="certificate-input"
                            autoResize
                            resizeOrientation="vertical"
                            id="config.keys"
                            value={(config.keys as string) || ''}
                            onChange={onChange}
                            isDisabled={isViewing || isActiveModificationsDisabled}
                            isRequired
                            placeholder={
                                '-----BEGIN CERTIFICATE-----\nAuthority certificate data\n-----END CERTIFICATE-----'
                            }
                            onBlur={onBlur}
                            validated={showUserPkiKeysError ? ValidatedOptions.error : 'default'}
                        />
                    </FormGroup>
                </GridItem>
            )}
            {type === 'iap' && (
                <GridItem span={12} lg={6}>
                    <FormGroup
                        label="Audience"
                        fieldId="config.audience"
                        isRequired
                        helperText={
                            <span className="pf-u-font-size-sm">
                                for example,{' '}
                                <kbd className="pf-u-font-size-xs">
                                    /projects/&lt;PROJECT_NUMBER&gt;/global/backendServices/&lt;SERVICE_ID&gt;
                                </kbd>
                            </span>
                        }
                        helperTextInvalid={configErrors?.audience || ''}
                        validated={showAudienceError ? ValidatedOptions.error : 'default'}
                    >
                        <TextInput
                            type="text"
                            id="config.audience"
                            value={(config.audience as string) || ''}
                            onChange={onChange}
                            isDisabled={isViewing || isActiveModificationsDisabled}
                            isRequired
                            onBlur={onBlur}
                            validated={showAudienceError ? ValidatedOptions.error : 'default'}
                        />
                    </FormGroup>
                </GridItem>
            )}
        </>
    );
}

export default ConfigurationFormFields;

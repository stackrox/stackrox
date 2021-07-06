import React, { ReactElement } from 'react';
import {
    Alert,
    Checkbox,
    FormGroup,
    GridItem,
    SelectOption,
    TextArea,
    TextInput,
} from '@patternfly/react-core';

import { oidcCallbackModes } from 'constants/accessControl';
import { AuthProviderType, AuthProviderConfig } from 'services/AuthService';
import SelectSingle from 'Components/SelectSingle'; // TODO import from where?

export type ConfigurationFormFieldsProps = {
    config: AuthProviderConfig;
    isViewing: boolean;
    onChange: (
        _value: unknown,
        event: React.FormEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>
    ) => void;
    setFieldValue: (name: string, value: string) => void;
    type: AuthProviderType;
};

const baseURL = `${window.location.protocol}//${window.location.host}`;
const oidcFragmentCallbackURL = `${baseURL}/auth/response/oidc`;
const oidcPostCallbackURL = `${baseURL}/sso/providers/oidc/callback`;
const samlACSURL = `${baseURL}/sso/providers/saml/acs`;

function ConfigurationFormFields({
    isViewing,
    onChange,
    config,
    setFieldValue,
    type,
}: ConfigurationFormFieldsProps): ReactElement {
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
                        >
                            <TextInput
                                type="text"
                                id="config.issuer"
                                value={(config.issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing}
                                isRequired
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup label="Client ID" fieldId="config.client_id" isRequired>
                            <TextInput
                                type="text"
                                id="config.client_id"
                                value={(config.client_id as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing}
                                isRequired
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
                                handleSelect={setFieldValue}
                                isDisabled={isViewing}
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
                        >
                            <TextInput
                                type="text"
                                id="config.issuer"
                                value={(config.issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing}
                                isRequired
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup label="Client ID" fieldId="config.client_id" isRequired>
                            <TextInput
                                type="text"
                                id="config.client_id"
                                value={(config.client_id as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing}
                                isRequired
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
                        <FormGroup
                            label="Client Secret"
                            fieldId="config.client_secret"
                            isRequired={config.mode !== 'fragment'}
                            helperText={
                                <span className="pf-u-font-size-sm">
                                    Client Secret provided by your IdP
                                </span>
                            }
                        >
                            <TextInput
                                type="text"
                                id="config.client_secret"
                                value={(config.client_secret as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing || config.mode === 'fragment'}
                                isRequired={config.mode !== 'fragment'}
                                placeholder={
                                    config.mode === 'fragment'
                                        ? 'Client Secret is not supported with Fragment callback mode.'
                                        : ''
                                }
                            />
                        </FormGroup>
                    </GridItem>
                    <GridItem span={12} lg={6}>
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
                                isDisabled={config.mode === 'query'}
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
                        >
                            <TextInput
                                type="text"
                                id="config.sp_issuer"
                                value={(config.sp_issuer as string) || ''}
                                onChange={onChange}
                                isDisabled={isViewing}
                                isRequired
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
                                isDisabled={isViewing}
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
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_metadata_url"
                                        value={(config.idp_metadata_url as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing}
                                        isRequired
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
                                        </span>
                                    }
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_issuer"
                                        value={(config.idp_issuer as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing}
                                        isRequired={config.configurationType === 'static'}
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
                                >
                                    <TextInput
                                        type="text"
                                        id="config.idp_sso_url"
                                        value={(config.idp_sso_url as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing}
                                        isRequired={config.configurationType === 'static'}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="Name/ID Format"
                                    fieldId="config.idp_nameid_format"
                                    isRequired={config.configurationType === 'static'}
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
                                        isDisabled={isViewing}
                                        isRequired={config.configurationType === 'static'}
                                    />
                                </FormGroup>
                            </GridItem>
                            <GridItem span={12} lg={6}>
                                <FormGroup
                                    label="IdP Certificate(s) (PEM)"
                                    fieldId="config.idp_cert_pem"
                                    isRequired={config.configurationType === 'static'}
                                >
                                    <TextArea
                                        autoResize
                                        resizeOrientation="vertical"
                                        id="config.idp_cert_pem"
                                        value={(config.idp_cert_pem as string) || ''}
                                        onChange={onChange}
                                        isDisabled={isViewing}
                                        isRequired={config.configurationType === 'static'}
                                        placeholder={
                                            '-----BEGIN CERTIFICATE-----\nYour certificate data\n-----END CERTIFICATE-----'
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
                        label="IdP Certificate(s) (PEM)"
                        fieldId="config.idp_cert_pem"
                        isRequired={config.configurationType === 'static'}
                    >
                        <TextArea
                            autoResize
                            resizeOrientation="vertical"
                            id="config.idp_cert_pem"
                            value={(config.idp_cert_pem as string) || ''}
                            onChange={onChange}
                            isDisabled={isViewing}
                            isRequired={config.configurationType === 'static'}
                            placeholder={
                                '-----BEGIN CERTIFICATE-----\nAuthority certificate data\n-----END CERTIFICATE-----'
                            }
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
                    >
                        <TextInput
                            type="text"
                            id="config.audience"
                            value={(config.audience as string) || ''}
                            onChange={onChange}
                            isDisabled={isViewing}
                            isRequired
                        />
                    </FormGroup>
                </GridItem>
            )}
        </>
    );
}

export default ConfigurationFormFields;

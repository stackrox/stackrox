import React, { ReactElement } from 'react';
import { FormGroup, GridItem, TextInput } from '@patternfly/react-core';

export type ConfigurationFormFieldsProps = {
    config: AuthProviderConfig;
    isViewing: boolean;
    onChange: (_value: unknown, event: React.FormEvent<HTMLInputElement>) => void;
};

export type AuthProviderConfig = Record<string, string | number | undefined | boolean>;

function ConfigurationFormFields({
    isViewing,
    onChange,
    config,
}: ConfigurationFormFieldsProps): ReactElement {
    return (
        <>
            <GridItem span={12} lg={6}>
                <FormGroup label="Auth0 tenant" fieldId="name" isRequired>
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
                <FormGroup label="Client ID" fieldId="name" isRequired>
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
        </>
    );
}

export default ConfigurationFormFields;

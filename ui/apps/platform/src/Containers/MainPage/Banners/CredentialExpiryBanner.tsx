import React, { ReactElement, useEffect, useState } from 'react';
import { Banner, Button } from '@patternfly/react-core';

import { generateCertSecretForComponent } from 'services/CertGenerationService';
import { fetchCertExpiryForComponent } from 'services/CredentialExpiryService';
import { CertExpiryComponent } from 'types/credentialExpiryService.proto';
import {
    getCredentialExpiryPhrase,
    getCredentialExpiryVariant,
    nameOfComponent,
} from 'utils/credentialExpiry';

type CredentialExpiryProps = {
    component: CertExpiryComponent;
    hasAdministrationWritePermission: boolean;
};

function CredentialExpiryBanner({
    component,
    hasAdministrationWritePermission,
}: CredentialExpiryProps): ReactElement | null {
    const [expirationDate, setExpirationDate] = useState('');
    useEffect(() => {
        fetchCertExpiryForComponent(component)
            .then((expiry) => {
                setExpirationDate(expiry);
            })
            .catch((e) => {
                // ignored because it's either a temporary network issue,
                //   or symptom of a larger problem
                // Either way, we don't want to spam the logimbue service

                // eslint-disable-next-line no-console
                console.warn(`Failed to fetch certification expiration for ${component}`, e);
            });
    }, [component]);

    if (!expirationDate) {
        return null;
    }
    const now = new Date(); // is this an impure side effect?
    const type = getCredentialExpiryVariant(expirationDate, now);
    if (type === 'success') {
        return null;
    }
    const downloadLink = (
        <Button variant="link" isInline onClick={() => generateCertSecretForComponent(component)}>
            download this YAML file
        </Button>
    );
    const name = nameOfComponent[component];
    const message = (
        <span className="flex-1 text-center">
            {`${name} certificate ${getCredentialExpiryPhrase(expirationDate, now)}. `}
            {hasAdministrationWritePermission ? (
                <>To use renewed certificates, {downloadLink} and apply it to your cluster.</>
            ) : (
                'Contact your administrator.'
            )}
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" variant={type}>
            {message}
        </Banner>
    );
}

export default CredentialExpiryBanner;

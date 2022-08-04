import React, { ReactElement, useEffect, useState } from 'react';
import { differenceInDays, distanceInWordsStrict, format } from 'date-fns';
import { Banner, Button } from '@patternfly/react-core';

import { generateCertSecretForComponent } from 'services/CertGenerationService';
import { fetchCertExpiryForComponent } from 'services/CredentialExpiryService';
import { CertExpiryComponent } from 'types/credentialExpiryService.proto';

function getExpirationMessageType(daysLeft: number): 'info' | 'danger' | 'warning' {
    if (daysLeft > 14) {
        return 'info';
    }
    if (daysLeft > 3) {
        return 'warning';
    }
    return 'danger';
}

const nameOfComponent: Record<CertExpiryComponent, string> = {
    CENTRAL: 'Central',
    SCANNER: 'Scanner',
};

type CredentialExpiryProps = {
    component: CertExpiryComponent;
    hasServiceIdentityWritePermission: boolean;
};

function CredentialExpiryBanner({
    component,
    hasServiceIdentityWritePermission,
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
    });

    if (!expirationDate) {
        return null;
    }
    const now = new Date(); // is this an impure side effect?
    const type = getExpirationMessageType(differenceInDays(expirationDate, now));
    if (type === 'info') {
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
            {name} certificate expires in {distanceInWordsStrict(expirationDate, now)} on{' '}
            {format(expirationDate, 'MMMM D, YYYY')} (at {format(expirationDate, 'h:mm a')}).{' '}
            {hasServiceIdentityWritePermission ? (
                <>To use renewed certificates, {downloadLink} and apply it to your cluster.</>
            ) : (
                'Contact your administrator.'
            )}
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" isSticky variant={type}>
            {message}
        </Banner>
    );
}

export default CredentialExpiryBanner;

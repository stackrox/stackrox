import React, { ReactElement } from 'react';
import { AlertVariant, Banner } from '@patternfly/react-core';

function DatabaseUnavailableBanner(): ReactElement {
    const message = (
        <span className="pf-u-text-align-center">
            The database is currently not available. If this problem persists, please contact
            support.
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" isSticky variant={AlertVariant.danger}>
            {message}
        </Banner>
    );
}

export default DatabaseUnavailableBanner;

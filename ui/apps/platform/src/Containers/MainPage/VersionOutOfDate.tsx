import React, { ReactElement } from 'react';
import { Banner, Button } from '@patternfly/react-core';

function refreshWindow() {
    window.location.reload();
}

function VersionOutOfDate(): ReactElement {
    const refreshButton = (
        <Button variant="link" isInline onClick={refreshWindow}>
            refresh this page
        </Button>
    );
    const message = (
        <span>
            It looks like this page is out of date and may not behave properly. Please{' '}
            {refreshButton} to correct any issues.
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" isSticky variant="warning">
            {message}
        </Banner>
    );
}

export default VersionOutOfDate;

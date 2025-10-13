import React from 'react';
import type { ReactElement } from 'react';
import { Banner, Button } from '@patternfly/react-core';

import useMetadata from 'hooks/useMetadata';

function reloadWindow() {
    window.location.reload();
}

function OutdatedVersionBanner(): ReactElement | null {
    const { isOutdatedVersion } = useMetadata();

    if (isOutdatedVersion) {
        return (
            <Banner className="pf-v5-u-text-align-center" variant="gold">
                It looks like this page is out of date and may not behave properly. Please{' '}
                <Button variant="link" isInline onClick={reloadWindow}>
                    refresh this page
                </Button>{' '}
                to correct any issues.
            </Banner>
        );
    }

    return null;
}

export default OutdatedVersionBanner;

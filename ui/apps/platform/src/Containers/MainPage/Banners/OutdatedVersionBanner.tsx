import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Banner, Button } from '@patternfly/react-core';

import { selectors } from 'reducers';

function reloadWindow() {
    window.location.reload();
}

function OutdatedVersionBanner(): ReactElement | null {
    const isOutdatedVersion = useSelector(selectors.isOutdatedVersionSelector);

    if (isOutdatedVersion) {
        return (
            <Banner className="pf-u-text-align-center" variant="warning">
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

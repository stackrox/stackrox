import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Banner, Button } from '@patternfly/react-core';

import { selectors } from 'reducers';

const reloadPage = () => window.location.reload();

const reloadPageButton = (
    <Button variant="link" isInline onClick={reloadPage}>
        refresh the page
    </Button>
);

function ServerStatusBanner(): ReactElement | null {
    const serverStatus = useSelector(selectors.serverStatusSelector);

    if (serverStatus === 'RESURRECTED') {
        return (
            <Banner className="pf-u-text-align-center" variant="success">
                The server has become reachable again after a connection problem. If you experience
                issues, please {reloadPageButton}
            </Banner>
        );
    }

    if (serverStatus === 'UNREACHABLE') {
        return (
            <Banner className="pf-u-text-align-center" variant="danger">
                There seems to be an issue reaching the server. Please check your network connection
                or {reloadPageButton}
            </Banner>
        );
    }

    return null;
}

export default ServerStatusBanner;

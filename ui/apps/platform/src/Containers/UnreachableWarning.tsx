import React, { ReactElement } from 'react';
import { Banner, Button } from '@patternfly/react-core';

import { serverStates } from 'reducers/serverError';

const refreshPage = () => window.location.reload();

export type ServerState = 'UP' | 'UNREACHABLE' | 'RESURRECTED' | undefined | null;
export type UnreachableWarningProps = {
    serverState: ServerState;
};

function UnreachableWarning({ serverState }: UnreachableWarningProps): ReactElement | null {
    if (!serverState || serverState === 'UP') {
        return null;
    }

    const message = (
        <span>
            {serverState === serverStates.UNREACHABLE &&
                `There seems to be an issue reaching the server. Please check your network connection or `}
            {serverState === serverStates.RESURRECTED &&
                `The server has become reachable again after a connection problem. If you experience issues, please `}
            <Button variant="link" isInline onClick={refreshPage}>
                refresh the page
            </Button>
            .
        </span>
    );

    return (
        <Banner className="pf-u-text-align-center" isSticky variant="warning">
            {message}
        </Banner>
    );
}

export default UnreachableWarning;

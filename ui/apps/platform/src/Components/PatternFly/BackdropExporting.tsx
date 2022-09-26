import React, { ReactElement } from 'react';
import { Backdrop, Bullseye, Spinner } from '@patternfly/react-core';

function BackdropExporting(): ReactElement {
    return (
        <Backdrop>
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        </Backdrop>
    );
}

export default BackdropExporting;

import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigHeader(): ReactElement | null {
    const publicConfigHeader = useSelector(selectors.publicConfigHeaderSelector);

    if (publicConfigHeader?.enabled) {
        return (
            <div
                className="pf-c-banner pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                style={getPublicConfigStyle(publicConfigHeader)}
                data-testid="public-config-header"
            >
                {publicConfigHeader.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigHeader;

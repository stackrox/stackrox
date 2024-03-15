import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigHeader(): ReactElement | null {
    const publicConfigHeader = useSelector(selectors.publicConfigHeaderSelector);

    if (publicConfigHeader?.enabled) {
        return (
            <div
                className="pf-v5-c-banner pf-v5-u-display-flex pf-v5-u-justify-content-center pf-v5-u-align-items-center"
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

import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigFooter(): ReactElement | null {
    const publicConfigFooter = useSelector(selectors.publicConfigFooterSelector);

    if (publicConfigFooter?.enabled) {
        return (
            <div
                className="pf-c-banner pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                style={getPublicConfigStyle(publicConfigFooter)}
                data-testid="public-config-footer"
            >
                {publicConfigFooter.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigFooter;

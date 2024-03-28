import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigFooter(): ReactElement | null {
    const publicConfigFooter = useSelector(selectors.publicConfigFooterSelector);

    if (publicConfigFooter?.enabled) {
        return (
            <div
                className="pf-v5-c-banner pf-v5-u-display-flex pf-v5-u-justify-content-center pf-v5-u-align-items-center"
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

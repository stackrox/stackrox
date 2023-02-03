import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';
import { BannerConfig } from 'types/config.proto';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigFooter(): ReactElement | null {
    const footer = useSelector<unknown, BannerConfig | null>(selectors.getPublicConfigFooter);

    if (footer?.enabled) {
        return (
            <div
                className="pf-c-banner pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                style={getPublicConfigStyle(footer)}
                data-testid="public-config-footer"
            >
                {footer.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigFooter;

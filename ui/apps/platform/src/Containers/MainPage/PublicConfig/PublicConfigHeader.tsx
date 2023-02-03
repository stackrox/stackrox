import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';
import { BannerConfig } from 'types/config.proto';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigHeader(): ReactElement | null {
    const header = useSelector<unknown, BannerConfig | null>(selectors.getPublicConfigHeader);

    if (header?.enabled) {
        return (
            <div
                className="pf-c-banner pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                style={getPublicConfigStyle(header)}
                data-testid="public-config-header"
            >
                {header.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigHeader;

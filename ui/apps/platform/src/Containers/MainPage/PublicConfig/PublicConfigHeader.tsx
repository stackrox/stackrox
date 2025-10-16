import React from 'react';
import type { ReactElement } from 'react';

import usePublicConfig from 'hooks/usePublicConfig';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigHeader(): ReactElement | null {
    const { publicConfig } = usePublicConfig();

    if (publicConfig?.header?.enabled) {
        return (
            <div
                className="pf-v5-c-banner pf-v5-u-display-flex pf-v5-u-justify-content-center pf-v5-u-align-items-center"
                style={getPublicConfigStyle(publicConfig.header)}
                data-testid="public-config-header"
            >
                {publicConfig.header.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigHeader;

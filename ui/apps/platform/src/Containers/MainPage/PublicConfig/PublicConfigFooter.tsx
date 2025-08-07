import React, { ReactElement } from 'react';

import usePublicConfig from 'hooks/usePublicConfig';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigFooter(): ReactElement | null {
    const { publicConfig } = usePublicConfig();

    if (publicConfig?.footer?.enabled) {
        return (
            <div
                className="pf-v5-c-banner pf-v5-u-display-flex pf-v5-u-justify-content-center pf-v5-u-align-items-center"
                style={getPublicConfigStyle(publicConfig.footer)}
                data-testid="public-config-footer"
            >
                {publicConfig.footer.text}
            </div>
        );
    }

    return null;
}

export default PublicConfigFooter;

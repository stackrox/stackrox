import type { ReactElement } from 'react';

import usePublicConfig from 'hooks/usePublicConfig';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigHeader(): ReactElement | null {
    const { publicConfig } = usePublicConfig();

    if (publicConfig?.header?.enabled) {
        return (
            <div
                className="pf-v6-c-banner pf-v6-u-display-flex pf-v6-u-justify-content-center pf-v6-u-align-items-center"
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

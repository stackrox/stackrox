import type { ReactElement } from 'react';

import usePublicConfig from 'hooks/usePublicConfig';

import { getPublicConfigStyle } from './PublicConfig.utils';

function PublicConfigFooter(): ReactElement | null {
    const { publicConfig } = usePublicConfig();

    if (publicConfig?.footer?.enabled) {
        return (
            <div
                className="pf-v6-c-banner pf-v6-u-display-flex pf-v6-u-justify-content-center pf-v6-u-align-items-center"
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

import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { clustersInitBundlesPath, integrationsPath } from 'routePaths';

function ManageTokensButton(): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isMoveInitBundlesEnabled = isFeatureFlagEnabled('ROX_MOVE_INIT_BUNDLES_UI');

    return (
        <HashLink
            to={
                isMoveInitBundlesEnabled
                    ? clustersInitBundlesPath
                    : `${integrationsPath}#token-integrations`
            }
            className="no-underline flex-shrink-0"
            data-testid="manageTokens"
        >
            {isMoveInitBundlesEnabled ? 'Manage init bundles' : 'Manage tokens'}
        </HashLink>
    );
}

export default ManageTokensButton;

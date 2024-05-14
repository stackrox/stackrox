import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';

import { clustersInitBundlesPath } from 'routePaths';

function ManageTokensButton(): ReactElement {
    return (
        <HashLink
            to={clustersInitBundlesPath}
            className="no-underline flex-shrink-0"
            data-testid="manageTokens"
        >
            Init bundles
        </HashLink>
    );
}

export default ManageTokensButton;

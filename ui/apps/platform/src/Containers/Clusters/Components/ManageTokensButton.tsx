import React from 'react';
import { integrationsPath } from 'routePaths';
import { HashLink } from 'react-router-hash-link';

const ManageTokensButton = () => (
    <HashLink
        to={`${integrationsPath}#token-integrations`}
        className="no-underline flex-shrink-0"
        data-testid="manageTokens"
    >
        Manage Tokens
    </HashLink>
);

export default ManageTokensButton;

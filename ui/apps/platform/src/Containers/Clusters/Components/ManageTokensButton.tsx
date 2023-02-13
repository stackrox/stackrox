import React from 'react';
import { integrationsPath } from 'routePaths';
import { HashLink } from 'react-router-hash-link';

const ManageTokensButton = () => (
    <HashLink
        to={`${integrationsPath}#token-integrations`}
        className="no-underline btn btn-base flex-shrink-0"
    >
        Manage Tokens
    </HashLink>
);

export default ManageTokensButton;

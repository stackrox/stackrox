import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Button, ButtonVariant, Spinner } from '@patternfly/react-core';

import { getEntityPath } from 'Containers/AccessControl/accessControlPaths';
import useFetchScopes from 'hooks/useFetchScopes';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type ScopeNameProps = {
    scopeId: string;
};

function ScopeName({ scopeId }: ScopeNameProps): ReactElement {
    const scopesResult = useFetchScopes();

    if (!scopeId) {
        return <em>No resource scope specified</em>;
    }

    const fullScope = scopesResult.scopes.find((scope) => scope.id === scopeId);

    if (scopesResult.isLoading) {
        return <Spinner isSVG size="md" />;
    }

    if (scopesResult.error) {
        return <span>Error getting scope info. {getAxiosErrorMessage(scopesResult.error)}</span>;
    }

    const url = getEntityPath('ACCESS_SCOPE', scopeId);

    return (
        <Button
            variant={ButtonVariant.link}
            isInline
            component={(props) => <Link {...props} to={url} />}
        >
            {fullScope?.name || scopeId}
        </Button>
    );
}

export default ScopeName;

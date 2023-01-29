import React, { ReactElement } from 'react';
import { Button, ButtonVariant, Spinner } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getEntityPath } from 'Containers/AccessControl/accessControlPaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useFetchScopes from 'hooks/useFetchScopes';
import useCollection from 'Containers/Collections/hooks/useCollection';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { collectionsBasePath } from 'routePaths';

type ScopeNameProps = {
    scopeId: string;
};

function AccessScopeName({ scopeId }: ScopeNameProps): ReactElement {
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
        <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
            {fullScope?.name || scopeId}
        </Button>
    );
}

function CollectionScopeName({ scopeId }: ScopeNameProps): ReactElement {
    const { data, loading, error } = useCollection(scopeId);

    if (!scopeId) {
        return <em>No report scope specified</em>;
    }

    if (loading) {
        return <Spinner isSVG size="md" />;
    }

    if (error) {
        return <span>Error getting scope info. {getAxiosErrorMessage(error)}</span>;
    }

    const url = `${collectionsBasePath}/${scopeId}`;

    return (
        <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
            {data?.collection?.name || scopeId}
        </Button>
    );
}

function ScopeName(props: ScopeNameProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isCollectionsEnabled = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    return isCollectionsEnabled ? (
        <CollectionScopeName {...props} />
    ) : (
        <AccessScopeName {...props} />
    );
}

export default ScopeName;

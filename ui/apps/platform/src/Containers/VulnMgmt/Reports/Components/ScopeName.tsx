import React, { ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import { Alert, Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getEntityPath } from 'Containers/AccessControl/accessControlPaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { collectionsBasePath } from 'routePaths';
import { ReportScope } from 'hooks/useFetchReport';

type ScopeNameProps = {
    reportScope: ReportScope | null;
    canWriteReports: boolean;
};

function AccessScopeName({ reportScope }: ScopeNameProps): ReactElement {
    if (!reportScope) {
        return <em>No resource scope specified</em>;
    }

    const url = getEntityPath('ACCESS_SCOPE', reportScope.id);

    return (
        <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
            {reportScope.name}
        </Button>
    );
}

function CollectionScopeName({ reportScope, canWriteReports }: ScopeNameProps): ReactElement {
    const history = useHistory();

    if (!reportScope) {
        return <em>No report scope specified</em>;
    }

    if (reportScope.type === 'AccessControlScope') {
        return (
            <Alert
                isInline
                variant="danger"
                title="The report scope for this configuration could not be migrated to a collection"
                actionLinks={
                    canWriteReports && (
                        <Button
                            variant={ButtonVariant.link}
                            isInline
                            component={LinkShim}
                            href={`${history.location.pathname as string}?action=edit`}
                        >
                            Edit this report
                        </Button>
                    )
                }
            />
        );
    }

    const url = `${collectionsBasePath}/${reportScope.id}`;

    return (
        <Button variant={ButtonVariant.link} isInline component={LinkShim} href={url}>
            {reportScope.name}
        </Button>
    );
}

function ScopeName(props: ScopeNameProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isCollectionsEnabled = isFeatureFlagEnabled('ROX_OBJECT_COLLECTIONS');

    return isCollectionsEnabled ? (
        <CollectionScopeName {...props} />
    ) : (
        <AccessScopeName {...props} />
    );
}

export default ScopeName;

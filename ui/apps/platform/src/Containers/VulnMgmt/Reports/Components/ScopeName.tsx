import React, { ReactElement } from 'react';
import { Alert, Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsBasePath } from 'routePaths';
import { ReportScope } from 'hooks/useFetchReport';

type ScopeNameProps = {
    reportScope: ReportScope | null;
};

function ScopeName({ reportScope }: ScopeNameProps): ReactElement {
    if (!reportScope) {
        return <em>No report scope specified</em>;
    }

    // TODO verify whether AccessControlScope can be deleted.
    if (reportScope.type === 'AccessControlScope') {
        return (
            <Alert
                isInline
                variant="danger"
                title="The report scope for this configuration could not be migrated to a collection"
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

export default ScopeName;

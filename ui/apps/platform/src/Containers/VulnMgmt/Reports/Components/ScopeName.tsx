import React, { ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import { Alert, Button, ButtonVariant } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsBasePath } from 'routePaths';
import { ReportScope } from 'hooks/useFetchReport';

type ScopeNameProps = {
    reportScope: ReportScope | null;
    canWriteReports: boolean;
};

function ScopeName({ reportScope, canWriteReports }: ScopeNameProps): ReactElement {
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

export default ScopeName;

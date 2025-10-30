import React from 'react';
import type { ReactElement } from 'react';
import { List, ListItem, Title } from '@patternfly/react-core';

import type { DryRunAlert } from 'services/PoliciesService';

type PreviewViolationsProps = {
    alertsFromDryRun: DryRunAlert[];
};

function PreviewViolations({ alertsFromDryRun }: PreviewViolationsProps): ReactElement {
    if (alertsFromDryRun.length === 0) {
        return <div>No deployments have violations.</div>;
    }

    return (
        <div>
            <Title className="pf-v5-u-mb-sm" headingLevel="h2">
                Deployment results
            </Title>
            {alertsFromDryRun.map(({ deployment, violations }, alertIndex) => {
                /*
                 * pf-v5-u-mb-sm separates deployment name from first list item with same spacing as subsequent list items.
                 * pf-v5-u-mt-mg separates subsequent deployment names with same spacing as bottom of explanation text.
                 */
                const className =
                    alertIndex === 0 ? 'pf-v5-u-mb-sm' : 'pf-v5-u-mb-sm pf-v5-u-mt-md';

                return (
                    // eslint-disable-next-line react/no-array-index-key
                    <div key={alertIndex}>
                        <Title headingLevel="h3" className={className}>
                            {deployment}
                        </Title>
                        <List>
                            {violations.map((violation, violationIndex) => (
                                // eslint-disable-next-line react/no-array-index-key
                                <ListItem key={violationIndex}>{violation}</ListItem>
                            ))}
                        </List>
                    </div>
                );
            })}
        </div>
    );
}

export default PreviewViolations;

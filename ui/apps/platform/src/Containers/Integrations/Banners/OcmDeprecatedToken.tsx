import React, { ReactElement } from 'react';
import { Alert, PageSection } from '@patternfly/react-core';
import { useSelector } from 'react-redux';

import { selectors } from 'reducers';

function ocmDeprecatedCounter(integrations: { type: string; credentials: { secret: string } }[]) {
    return () =>
        integrations.filter(
            (integration) =>
                integration.type.toLowerCase() === 'type_ocm' && integration.credentials.secret
        ).length;
}

function OcmDeprecatedTokenBanner(): ReactElement | null {
    const integrations = useSelector(selectors.getCloudSources);
    const countIntegrations = ocmDeprecatedCounter(integrations);

    if (countIntegrations() === 0) {
        return null;
    }
    return (
        <PageSection variant="light">
            <Alert
                variant="warning"
                component="p"
                title="Deprecated cloud source configuration found"
            >
                <p>
                    A OpenShift Cluster Manager integration has been configured with the deprecated
                    API token option. Use service account authentication instead.
                </p>
            </Alert>
        </PageSection>
    );
}

export default OcmDeprecatedTokenBanner;

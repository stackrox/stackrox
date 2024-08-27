import React, { ReactElement } from 'react';
import { Alert } from '@patternfly/react-core';
import { useSelector } from 'react-redux';
import { CloudSourceIntegration } from 'services/CloudSourceService';

import { selectors } from 'reducers';

function ocmDeprecatedCounter(integrations: CloudSourceIntegration[]) {
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
        <Alert
            isInline
            variant="warning"
            component="p"
            title="Deprecated cloud source configuration found"
        >
            <p>
                An OpenShift Cluster Manager integration has been configured with the deprecated API
                token option. Use service account authentication instead.
            </p>
        </Alert>
    );
}

export default OcmDeprecatedTokenBanner;

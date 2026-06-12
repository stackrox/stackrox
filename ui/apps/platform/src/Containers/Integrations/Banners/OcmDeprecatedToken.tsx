import type { ReactElement } from 'react';
import { Alert } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchCloudSources } from 'services/CloudSourceService';
import type { CloudSourceIntegration } from 'services/CloudSourceService';

function fetchCloudSourceList(): Promise<CloudSourceIntegration[]> {
    return fetchCloudSources().then((r) => r.cloudSources);
}

function ocmDeprecatedCounter(integrations: CloudSourceIntegration[]) {
    return () =>
        integrations.filter(
            (integration) =>
                integration.type.toLowerCase() === 'type_ocm' && integration.credentials.secret
        ).length;
}

function OcmDeprecatedToken(): ReactElement | null {
    const { data: integrations } = useRestQuery(fetchCloudSourceList);

    const deprecatedCount = ocmDeprecatedCounter(integrations ?? []);

    if (deprecatedCount() === 0) {
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

export default OcmDeprecatedToken;

import { useCallback } from 'react';
import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import useRestQuery from 'hooks/useRestQuery';
import { getDiscoveredScanConfiguration } from 'services/ComplianceScanConfigurationService';

import ViewScanConfigDetail from './ViewScanConfigDetail';

function DiscoveredScanConfigDetailPage(): ReactElement {
    const { scanConfigName } = useParams() as { scanConfigName: string };
    const decodedName = decodeURIComponent(scanConfigName);

    const scanConfigQuery = useCallback(
        () => getDiscoveredScanConfiguration(decodedName),
        [decodedName]
    );
    const { data: scanConfig, isLoading, error } = useRestQuery(scanConfigQuery);

    return (
        <ViewScanConfigDetail
            hasWriteAccessForCompliance={false}
            scanConfig={scanConfig}
            isLoading={isLoading}
            error={error}
        />
    );
}

export default DiscoveredScanConfigDetailPage;

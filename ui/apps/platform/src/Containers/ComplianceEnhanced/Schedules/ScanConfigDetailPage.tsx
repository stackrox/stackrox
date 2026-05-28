import { useCallback } from 'react';
import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import useRestQuery from 'hooks/useRestQuery';
import { getComplianceScanConfiguration } from 'services/ComplianceScanConfigurationService';
import ViewScanConfigDetail from './ViewScanConfigDetail';

type ScanConfigDetailPageProps = {
    hasWriteAccessForCompliance: boolean;
};

function ScanConfigDetailPage({
    hasWriteAccessForCompliance,
}: ScanConfigDetailPageProps): ReactElement {
    const { scanConfigId } = useParams() as { scanConfigId: string };

    const scanConfigFetcher = useCallback(() => {
        const { request, cancel } = getComplianceScanConfiguration(scanConfigId);
        return { request, cancel };
    }, [scanConfigId]);

    const { data, isLoading, error } = useRestQuery(scanConfigFetcher);

    return (
        <ViewScanConfigDetail
            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
            scanConfig={data}
            isLoading={isLoading}
            error={error}
        />
    );
}

export default ScanConfigDetailPage;

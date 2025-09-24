/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useCallback } from 'react';
import type { ReactElement } from 'react';
import { generatePath, Link, useParams } from 'react-router-dom-v5-compat';

import usePageAction from 'hooks/usePageAction';
import useRestQuery from 'hooks/useRestQuery';
import { getComplianceScanConfiguration } from 'services/ComplianceScanConfigurationService';
import EditScanConfigDetail from './EditScanConfigDetail';
import ViewScanConfigDetail from './ViewScanConfigDetail';
import type { PageActions } from './compliance.scanConfigs.utils';

type ScanConfigDetailPageProps = {
    hasWriteAccessForCompliance: boolean;
};

function ScanConfigDetailPage({
    hasWriteAccessForCompliance,
}: ScanConfigDetailPageProps): ReactElement {
    const { scanConfigId } = useParams() as { scanConfigId: string };
    const { pageAction } = usePageAction<PageActions>();

    const scanConfigFetcher = useCallback(() => {
        const { request, cancel } = getComplianceScanConfiguration(scanConfigId);
        return { request, cancel };
    }, [scanConfigId]);

    const { data, isLoading, error } = useRestQuery(scanConfigFetcher);

    if (pageAction === 'edit' && hasWriteAccessForCompliance) {
        return <EditScanConfigDetail scanConfig={data} isLoading={isLoading} error={error} />;
    }

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

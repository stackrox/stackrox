/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useCallback } from 'react';
import { generatePath, Link, useParams } from 'react-router-dom';

import usePageAction from 'hooks/usePageAction';
import useRestQuery from 'hooks/useRestQuery';
import { getComplianceScanConfiguration } from 'services/ComplianceScanConfigurationService';
import EditScanConfigDetail from './EditScanConfigDetail';
import ViewScanConfigDetail from './ViewScanConfigDetail';
import { PageActions } from './compliance.scanConfigs.utils';

type ScanConfigDetailPageProps = {
    hasWriteAccessForCompliance: boolean;
    isReportJobsEnabled: boolean;
    isComplianceReportingEnabled: boolean;
};

function ScanConfigDetailPage({
    hasWriteAccessForCompliance,
    isReportJobsEnabled,
    isComplianceReportingEnabled,
}: ScanConfigDetailPageProps): React.ReactElement {
    const { scanConfigId } = useParams();
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
            isReportJobsEnabled={isReportJobsEnabled}
            isComplianceReportingEnabled={isComplianceReportingEnabled}
            scanConfig={data}
            isLoading={isLoading}
            error={error}
        />
    );
}

export default ScanConfigDetailPage;

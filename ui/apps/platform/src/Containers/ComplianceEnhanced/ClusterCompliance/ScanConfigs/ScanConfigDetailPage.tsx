/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useCallback } from 'react';
import { generatePath, Link, useParams } from 'react-router-dom';
import { Button, Divider, PageSection } from '@patternfly/react-core';

import usePageAction from 'hooks/usePageAction';
import useRestQuery from 'hooks/useRestQuery';
import { getScanConfig } from 'services/ComplianceEnhancedService';
import EditScanConfigDetail from './EditScanConfigDetail';
import ViewScanConfigDetail from './ViewScanConfigDetail';
import { PageActions } from './compliance.scanConfigs.utils';

type ScanConfigDetailPageProps = {
    hasWriteAccessForCompliance: boolean;
};

function ScanConfigDetailPage({
    hasWriteAccessForCompliance,
}: ScanConfigDetailPageProps): React.ReactElement {
    const { scanConfigId } = useParams();
    const { pageAction } = usePageAction<PageActions>();

    const scanConfigFetcher = useCallback(() => {
        const { request, cancel } = getScanConfig(scanConfigId);
        return { request, cancel };
    }, [scanConfigId]);

    const { data, loading, error } = useRestQuery(scanConfigFetcher);

    if (pageAction === 'edit' && hasWriteAccessForCompliance) {
        return <EditScanConfigDetail scanConfig={data} isLoading={loading} error={error} />;
    }

    return (
        <ViewScanConfigDetail
            hasWriteAccessForCompliance={hasWriteAccessForCompliance}
            scanConfig={data}
            isLoading={loading}
            error={error}
        />
    );
}

export default ScanConfigDetailPage;

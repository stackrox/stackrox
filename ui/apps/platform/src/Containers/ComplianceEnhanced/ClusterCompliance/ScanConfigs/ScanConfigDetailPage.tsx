/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useCallback } from 'react';
import { generatePath, Link, useParams } from 'react-router-dom';
import { Button, Divider, PageSection } from '@patternfly/react-core';

import { complianceEnhancedScanConfigDetailPath } from 'routePaths';
import useRestQuery from 'hooks/useRestQuery';
import { getScanConfig } from 'services/ComplianceEnhancedService';
import ViewScanConfigDetail from './ViewScanConfigDetail';

type ScanConfigDetailPageProps = {
    hasWriteAccessForCompliance: boolean;
};

function ScanConfigDetailPage({
    hasWriteAccessForCompliance,
}: ScanConfigDetailPageProps): React.ReactElement {
    const { scanConfigId } = useParams();

    const scanConfigFetcher = useCallback(() => {
        const { request, cancel } = getScanConfig(scanConfigId);
        return { request, cancel };
    }, [scanConfigId]);

    const { data, loading, error } = useRestQuery(scanConfigFetcher);

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

import React from 'react';

import { complianceEnhancedCoveragePath, complianceEnhancedScanConfigsPath } from 'routePaths';
import TabNavHeader from 'Components/TabNav/TabNavHeader';

type ClusterComplianceHeaderProps = {
    currentTabTitle: string;
};

function ClusterComplianceHeader({ currentTabTitle }: ClusterComplianceHeaderProps) {
    const tabLinks = [
        { title: 'Coverage', href: complianceEnhancedCoveragePath },
        { title: 'Schedules', href: complianceEnhancedScanConfigsPath },
    ];

    return (
        <>
            <TabNavHeader
                currentTabTitle={currentTabTitle}
                tabLinks={tabLinks}
                pageTitle="Compliance - Cluster compliance"
                mainTitle="Cluster compliance"
            />
        </>
    );
}

export default ClusterComplianceHeader;

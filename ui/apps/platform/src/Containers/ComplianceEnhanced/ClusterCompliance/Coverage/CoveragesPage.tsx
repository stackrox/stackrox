import React from 'react';
import { Divider } from '@patternfly/react-core';

import TabNavHeader from 'Components/TabNav/TabNavHeader';
import { complianceEnhancedCoveragePath, complianceEnhancedScanConfigsPath } from 'routePaths';

function CoveragesPage() {
    return (
        <>
            <TabNavHeader
                currentTabTitle="Coverage"
                tabLinks={[
                    { title: 'Coverage', href: complianceEnhancedCoveragePath },
                    { title: 'Schedules', href: complianceEnhancedScanConfigsPath },
                ]}
                pageTitle="Compliance - Cluster compliance"
                mainTitle="Cluster compliance"
            />
            <Divider component="div" />
        </>
    );
}

export default CoveragesPage;

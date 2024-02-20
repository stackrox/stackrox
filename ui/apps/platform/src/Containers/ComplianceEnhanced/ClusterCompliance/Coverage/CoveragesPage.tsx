import React from 'react';
import { Button, Divider, PageSection } from '@patternfly/react-core';

import TabNavHeader from 'Components/TabNav/TabNavHeader';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import { complianceEnhancedCoveragePath, complianceEnhancedScanConfigsPath } from 'routePaths';
import useURLStringUnion from 'hooks/useURLStringUnion';

import ClustersCoverageTable from './ClustersCoverageTable';

function CoveragesPage() {
    const [activeEntityTabKey] = useURLStringUnion('tableView', ['Clusters', 'Profiles']);

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
            <TabNavSubHeader
                description="Assess profile compliance for platform resources and nodes across clusters"
                actions={
                    <Button isDisabled variant="primary">
                        Export data as CSV (coming soon)
                    </Button>
                }
            />
            <Divider component="div" />
            <PageSection>
                {activeEntityTabKey === 'Clusters' && <ClustersCoverageTable />}
            </PageSection>
        </>
    );
}

export default CoveragesPage;

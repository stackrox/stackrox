import React from 'react';
import { Button, Divider, PageSection } from '@patternfly/react-core';

import TabNavHeader from 'Components/TabNav/TabNavHeader';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import { complianceEnhancedCoveragePath, complianceEnhancedScanConfigsPath } from 'routePaths';
import useURLStringUnion from 'hooks/useURLStringUnion';

import CoverageTableViewToggleGroup from './Components/CoverageTableViewToggleGroup';
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
                actions={<Button variant="primary">Export data as CSV</Button>}
            />
            <Divider component="div" />
            <PageSection>
                <CoverageTableViewToggleGroup />
                <Divider component="div" />
                {activeEntityTabKey === 'Clusters' && <ClustersCoverageTable />}
            </PageSection>
        </>
    );
}

export default CoveragesPage;

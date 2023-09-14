import React, { useState } from 'react';
import { generatePath } from 'react-router-dom';

import WidgetCard from 'Components/PatternFly/WidgetCard';
import { complianceEnhancedStatusClustersPath } from 'routePaths';

import HorizontalBarChart from './HorizontalBarChart';

export type ComplianceByClusterData = {
    name: string;
    passing: number;
    link: string;
}[];

const mockComplianceData: ComplianceByClusterData = [
    {
        name: 'staging',
        passing: 100,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
    {
        name: 'production',
        passing: 80,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
    {
        name: 'payments',
        passing: 73,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
    {
        name: 'patient-charts',
        passing: 69,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
    {
        name: 'another-cluster',
        passing: 67,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
    {
        name: 'cluster-name',
        passing: 39,
        link: generatePath(complianceEnhancedStatusClustersPath, { id: '123456' }),
    },
];

function ComplianceByCluster() {
    const [complianceData] = useState(mockComplianceData);

    return (
        <WidgetCard isLoading={false} header="Compliance by cluster">
            <HorizontalBarChart passingRateData={complianceData} />
        </WidgetCard>
    );
}

export default ComplianceByCluster;

import React, { useState } from 'react';

import WidgetCard from 'Components/PatternFly/WidgetCard';

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
        link: '',
    },
    {
        name: 'production',
        passing: 80,
        link: '',
    },
    {
        name: 'payments',
        passing: 73,
        link: '',
    },
    {
        name: 'patient-charts',
        passing: 69,
        link: '',
    },
    {
        name: 'another-cluster',
        passing: 67,
        link: '',
    },
    {
        name: 'cluster-name',
        passing: 39,
        link: '',
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

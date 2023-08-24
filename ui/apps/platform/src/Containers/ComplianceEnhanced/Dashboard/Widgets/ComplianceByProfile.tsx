import React, { useState } from 'react';

import WidgetCard from 'Components/PatternFly/WidgetCard';

import HorizontalBarChart from './HorizontalBarChart';

export type ComplianceByProfileData = {
    name: string;
    passing: number;
    link: string;
}[];

const mockComplianceData: ComplianceByProfileData = [
    {
        name: 'HIPPA',
        passing: 83,
        link: '',
    },
    {
        name: 'PCI',
        passing: 80,
        link: '',
    },
    {
        name: 'CIS Docker',
        passing: 73,
        link: '',
    },
    {
        name: 'CIS K8s',
        passing: 69,
        link: '',
    },
];

function ComplianceByProfile() {
    const [complianceData] = useState(mockComplianceData);

    return (
        <WidgetCard isLoading={false} header="Compliance by profile">
            <HorizontalBarChart passingRateData={complianceData} />
        </WidgetCard>
    );
}

export default ComplianceByProfile;

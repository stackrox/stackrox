import React, { useState } from 'react';
import { generatePath } from 'react-router-dom';

import WidgetCard from 'Components/PatternFly/WidgetCard';
import { complianceEnhancedStatusProfilesPath } from 'routePaths';

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
        link: generatePath(complianceEnhancedStatusProfilesPath, { id: '123456' }),
    },
    {
        name: 'PCI',
        passing: 80,
        link: generatePath(complianceEnhancedStatusProfilesPath, { id: '123456' }),
    },
    {
        name: 'CIS K8s',
        passing: 69,
        link: generatePath(complianceEnhancedStatusProfilesPath, { id: '123456' }),
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

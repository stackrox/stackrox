import React from 'react';

import DashboardLayout from 'Components/DashboardLayout';

import PoliciesCountTile from './PoliciesCountTile';
import CvesCountTile from './CvesCountTile';
import ApplicationDashboardMenu from './ApplicationDashboardMenu';
import FilterCvesRadioButtonGroup from './FilterCvesRadioButtonGroup';

import TopRiskyEntitiesByVulnerabilities from '../widgets/TopRiskyEntitiesByVulnerabilities';
import TopRiskiestImagesAndComponents from '../widgets/TopRiskiestImagesAndComponents';
import FrequentlyViolatedPolicies from '../widgets/FrequentlyViolatedPolicies';
import MostRecentVulnerabilities from '../widgets/MostRecentVulnerabilities';
import MostCommonVulnerabilities from '../widgets/MostCommonVulnerabilities';
import DeploymentsWithMostSeverePolicyViolations from '../widgets/DeploymentsWithMostSeverePolicyViolations';
import ClustersWithMostK8sIstioVulnerabilities from '../widgets/ClustersWithMostK8sIstioVulnerabilities';

const VulnDashboardPage = () => {
    const headerComponents = (
        <>
            <PoliciesCountTile />
            <CvesCountTile />
            <ApplicationDashboardMenu />
            <FilterCvesRadioButtonGroup />
        </>
    );
    return (
        <DashboardLayout headerText="Vulnerability Management" headerComponents={headerComponents}>
            <TopRiskyEntitiesByVulnerabilities />
            <TopRiskiestImagesAndComponents />
            <FrequentlyViolatedPolicies />
            <MostRecentVulnerabilities />
            <MostCommonVulnerabilities />
            <DeploymentsWithMostSeverePolicyViolations />
            <ClustersWithMostK8sIstioVulnerabilities />
        </DashboardLayout>
    );
};
export default VulnDashboardPage;

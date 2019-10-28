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
import ClustersWithMostK8sVulnerabilities from '../widgets/ClustersWithMostK8sVulnerabilities';

// layout-specific graph widget counts
const recentVulnerabilitiesDashboardCount = 8;
const topRiskiestCount = 8;
const mostSeverePolicyVioloationsCount = 8;

const VulnDashboardPage = () => {
    const headerComponents = (
        <>
            <PoliciesCountTile />
            <CvesCountTile />
            <div className="flex w-32">
                <ApplicationDashboardMenu />
            </div>
            <FilterCvesRadioButtonGroup />
        </>
    );
    return (
        <DashboardLayout headerText="Vulnerability Management" headerComponents={headerComponents}>
            <div className="sx-4 sy-2">
                <TopRiskyEntitiesByVulnerabilities />
            </div>
            <div className="s-2">
                <TopRiskiestImagesAndComponents limit={topRiskiestCount} />
            </div>
            <div className="s-2">
                <FrequentlyViolatedPolicies />
            </div>
            <div className="s-2">
                <MostRecentVulnerabilities limit={recentVulnerabilitiesDashboardCount} />
            </div>
            <div className="sx-2 sy-4">
                <MostCommonVulnerabilities />
            </div>
            <div className="s-2">
                <DeploymentsWithMostSeverePolicyViolations
                    limit={mostSeverePolicyVioloationsCount}
                />
            </div>
            <div className="s-2">
                <ClustersWithMostK8sVulnerabilities />
            </div>
        </DashboardLayout>
    );
};
export default VulnDashboardPage;

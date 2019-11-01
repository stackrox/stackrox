import React, { useContext } from 'react';

import entityTypes from 'constants/entityTypes';
import DashboardLayout from 'Components/DashboardLayout';
import EntitiesMenu from 'Components/workflow/EntitiesMenu';
import workflowStateContext from 'Containers/workflowStateContext';

import { dashboardLimit } from 'constants/workflowPages.constants';
import PoliciesCountTile from './PoliciesCountTile';
import CvesCountTile from './CvesCountTile';
import FilterCvesRadioButtonGroup from './FilterCvesRadioButtonGroup';

import TopRiskyEntitiesByVulnerabilities from '../widgets/TopRiskyEntitiesByVulnerabilities';
import TopRiskiestImagesAndComponents from '../widgets/TopRiskiestImagesAndComponents';
import FrequentlyViolatedPolicies from '../widgets/FrequentlyViolatedPolicies';
import MostRecentVulnerabilities from '../widgets/MostRecentVulnerabilities';
import MostCommonVulnerabilities from '../widgets/MostCommonVulnerabilities';
import DeploymentsWithMostSeverePolicyViolations from '../widgets/DeploymentsWithMostSeverePolicyViolations';
import ClustersWithMostK8sVulnerabilities from '../widgets/ClustersWithMostK8sVulnerabilities';

// layout-specific graph widget counts

const VulnDashboardPage = () => {
    const workflowState = useContext(workflowStateContext);

    const searchState = workflowState.getCurrentSearchState();

    const entityMenuTypes = [
        entityTypes.CLUSTER,
        entityTypes.NAMESPACE,
        entityTypes.DEPLOYMENT,
        entityTypes.IMAGE,
        entityTypes.COMPONENT
    ];
    const headerComponents = (
        <>
            <PoliciesCountTile />
            <CvesCountTile />
            <div className="flex w-32">
                <EntitiesMenu text="Application & Infrastructure" options={entityMenuTypes} />
            </div>
            <FilterCvesRadioButtonGroup />
        </>
    );
    return (
        <DashboardLayout headerText="Vulnerability Management" headerComponents={headerComponents}>
            <div className="sx-4 sy-2">
                <TopRiskyEntitiesByVulnerabilities defaultSelection={entityTypes.DEPLOYMENT} />
            </div>
            <div className="s-2">
                <TopRiskiestImagesAndComponents limit={dashboardLimit} />
            </div>
            <div className="s-2">
                <FrequentlyViolatedPolicies />
            </div>
            <div className="s-2">
                <MostRecentVulnerabilities search={searchState} limit={dashboardLimit} />
            </div>
            <div className="sx-2 sy-4">
                <MostCommonVulnerabilities search={searchState} />
            </div>
            <div className="s-2">
                <DeploymentsWithMostSeverePolicyViolations limit={dashboardLimit} />
            </div>
            <div className="s-2">
                <ClustersWithMostK8sVulnerabilities />
            </div>
        </DashboardLayout>
    );
};
export default VulnDashboardPage;

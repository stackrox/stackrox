import React, { useContext } from 'react';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

import entityTypes from 'constants/entityTypes';
import { createOptions } from 'utils/workflowUtils';
import DashboardLayout from 'Components/DashboardLayout';
import ExportButton from 'Components/ExportButton';
import RadioButtonGroup from 'Components/RadioButtonGroup';
import workflowStateContext from 'Containers/workflowStateContext';
import { DASHBOARD_LIMIT } from 'constants/workflowPages.constants';
import DashboardMenu from 'Components/DashboardMenu';
import PoliciesCountTile from '../Components/PoliciesCountTile';
import CvesCountTile from '../Components/CvesCountTile';
import TopRiskyEntitiesByVulnerabilities from '../widgets/TopRiskyEntitiesByVulnerabilities';
import TopRiskiestImagesAndComponents from '../widgets/TopRiskiestImagesAndComponents';
import FrequentlyViolatedPolicies from '../widgets/FrequentlyViolatedPolicies';
import RecentlyDetectedVulnerabilities from '../widgets/RecentlyDetectedVulnerabilities';
import MostCommonVulnerabilities from '../widgets/MostCommonVulnerabilities';
import DeploymentsWithMostSeverePolicyViolations from '../widgets/DeploymentsWithMostSeverePolicyViolations';
import ClustersWithMostK8sIstioVulnerabilities from '../widgets/ClustersWithMostK8sIstioVulnerabilities';

const entityMenuTypes = [
    entityTypes.CLUSTER,
    entityTypes.NAMESPACE,
    entityTypes.DEPLOYMENT,
    entityTypes.IMAGE,
    entityTypes.COMPONENT
];

const VulnDashboardPage = ({ history }) => {
    const workflowState = useContext(workflowStateContext);
    const searchState = workflowState.getCurrentSearchState();

    const cveFilterButtons = [
        {
            text: 'Fixable'
        },
        {
            text: 'All'
        }
    ];

    function handleCveFilterToggle(value) {
        const selectedOption = cveFilterButtons.find(button => button.text === value);
        const newValue = selectedOption.text || 'All';

        let targetUrl;
        if (newValue === 'Fixable') {
            targetUrl = workflowState
                .setSearch({
                    Fixable: 'true'
                })
                .toUrl();
        } else {
            const allSearch = { ...searchState };
            delete allSearch.Fixable;

            targetUrl = workflowState.setSearch(allSearch).toUrl();
        }

        history.push(targetUrl);
    }

    const cveFilter = searchState.Fixable ? 'Fixable' : 'All';

    const headerComponents = (
        <>
            <div className="flex items-center">
                <RadioButtonGroup
                    buttons={cveFilterButtons}
                    headerText="Filter CVEs"
                    onClick={handleCveFilterToggle}
                    selected={cveFilter}
                />
                <ExportButton
                    fileName="Vulnerability Management Dashboard Report"
                    page={workflowState.useCase}
                    pdfId="capture-dashboard"
                />
            </div>
            <div className="flex h-full ml-3 pl-3 border-l border-base-400">
                <PoliciesCountTile />
                <CvesCountTile />
                <div className="flex w-32">
                    <DashboardMenu
                        text="Application & Infrastructure"
                        options={createOptions(entityMenuTypes, workflowState)}
                    />
                </div>
            </div>
        </>
    );
    return (
        <DashboardLayout headerText="Vulnerability Management" headerComponents={headerComponents}>
            <div className="s-2 md:sx-4 xxxl:sx-4 ">
                <TopRiskyEntitiesByVulnerabilities
                    defaultSelection={entityTypes.DEPLOYMENT}
                    cveFilter={cveFilter}
                />
            </div>
            <div className="s-2 xxxl:sx-2">
                <TopRiskiestImagesAndComponents limit={DASHBOARD_LIMIT} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <FrequentlyViolatedPolicies />
            </div>
            <div className="s-2 xxxl:sx-2">
                <RecentlyDetectedVulnerabilities search={searchState} limit={DASHBOARD_LIMIT} />
            </div>
            <div className="s-2 md:sy-2 md:sx-2 lg:sy-4 xxxl:sx-2">
                <MostCommonVulnerabilities search={searchState} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <DeploymentsWithMostSeverePolicyViolations limit={DASHBOARD_LIMIT} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <ClustersWithMostK8sIstioVulnerabilities />
            </div>
        </DashboardLayout>
    );
};

VulnDashboardPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired
};

export default withRouter(VulnDashboardPage);

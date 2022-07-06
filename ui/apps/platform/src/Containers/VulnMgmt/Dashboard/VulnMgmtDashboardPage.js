import React, { useContext } from 'react';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

import entityTypes from 'constants/entityTypes';
import { createOptions } from 'utils/workflowUtils';
import DashboardLayout from 'Components/DashboardLayout';
import ExportButton from 'Components/ExportButton';
import PageTitle from 'Components/PageTitle';
import RadioButtonGroup from 'Components/RadioButtonGroup';
import workflowStateContext from 'Containers/workflowStateContext';
import { DASHBOARD_LIMIT } from 'constants/workflowPages.constants';
import DashboardMenu from 'Components/DashboardMenu';
import useFeatureFlags from 'hooks/useFeatureFlags';
import PoliciesCountTile from '../Components/PoliciesCountTile';
import CvesCountTile from '../Components/CvesCountTile';
import ImagesCountTile from '../Components/ImagesCountTile';
import NodesCountTile from '../Components/NodesCountTile';
import TopRiskyEntitiesByVulnerabilities from '../widgets/TopRiskyEntitiesByVulnerabilities';
import TopRiskiestEntities from '../widgets/TopRiskiestEntities';
import FrequentlyViolatedPolicies from '../widgets/FrequentlyViolatedPolicies';
import RecentlyDetectedImageVulnerabilities from '../widgets/RecentlyDetectedImageVulnerabilities';
import MostCommonVulnerabilities from '../widgets/MostCommonVulnerabilities';
import DeploymentsWithMostSeverePolicyViolations from '../widgets/DeploymentsWithMostSeverePolicyViolations';
import ClustersWithMostOrchestratorIstioVulnerabilities from '../widgets/ClustersWithMostOrchestratorIstioVulnerabilities';

const baseEntityMenuTypes = [entityTypes.CLUSTER, entityTypes.NAMESPACE, entityTypes.DEPLOYMENT];
const componentMenuType = [entityTypes.COMPONENT];
const splitComponentMenuTypes = [entityTypes.NODE_COMPONENT, entityTypes.IMAGE_COMPONENT];

const VulnDashboardPage = ({ history }) => {
    const workflowState = useContext(workflowStateContext);
    const searchState = workflowState.getCurrentSearchState();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    let entityMenuTypes = [...baseEntityMenuTypes];
    if (isFeatureFlagEnabled('ROX_FRONTEND_VM_UDPATES')) {
        entityMenuTypes = [...baseEntityMenuTypes, ...splitComponentMenuTypes];
    } else {
        entityMenuTypes = [...baseEntityMenuTypes, ...componentMenuType];
    }

    const cveFilterButtons = [
        {
            text: 'Fixable',
        },
        {
            text: 'All',
        },
    ];

    function handleCveFilterToggle(value) {
        const selectedOption = cveFilterButtons.find((button) => button.text === value);
        const newValue = selectedOption.text || 'All';

        let targetUrl;
        if (newValue === 'Fixable') {
            targetUrl = workflowState
                .setSearch({
                    Fixable: 'true',
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
            <PageTitle title="Vulnerability Management - Dashboard" />
            <div className="flex items-center">
                <div className="flex h-full mr-3 pr-3 border-r-2 border-base-400">
                    <PoliciesCountTile />
                    <CvesCountTile />
                    <NodesCountTile />
                    <ImagesCountTile />
                    <div className="flex w-32">
                        <DashboardMenu
                            text="Application & Infrastructure"
                            options={createOptions(entityMenuTypes, workflowState)}
                        />
                    </div>
                </div>
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
                <TopRiskiestEntities limit={DASHBOARD_LIMIT} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <FrequentlyViolatedPolicies />
            </div>
            <div className="s-2 xxxl:sx-2">
                <RecentlyDetectedImageVulnerabilities
                    search={searchState}
                    limit={DASHBOARD_LIMIT}
                />
            </div>
            <div className="s-2 md:sy-2 md:sx-2 lg:sy-4 xxxl:sx-2">
                <MostCommonVulnerabilities search={searchState} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <DeploymentsWithMostSeverePolicyViolations limit={DASHBOARD_LIMIT} />
            </div>
            <div className="s-2 xxxl:sx-2">
                <ClustersWithMostOrchestratorIstioVulnerabilities />
            </div>
        </DashboardLayout>
    );
};

VulnDashboardPage.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
};

export default withRouter(VulnDashboardPage);

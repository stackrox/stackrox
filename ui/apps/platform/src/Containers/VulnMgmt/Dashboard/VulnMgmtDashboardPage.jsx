import React, { useContext } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';

import usePermissions from 'hooks/usePermissions';
import entityTypes from 'constants/entityTypes';
import { createOptions } from 'utils/workflowUtils';
import DashboardLayout from 'Components/DashboardLayout';
import PageTitle from 'Components/PageTitle';
import RadioButtonGroup from 'Components/RadioButtonGroup';
import workflowStateContext from 'Containers/workflowStateContext';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import { DASHBOARD_LIMIT } from 'constants/workflowPages.constants';
import DashboardMenu from 'Components/DashboardMenu';
import ImagesCountTile from '../Components/ImagesCountTile';
import NodesCountTile from '../Components/NodesCountTile';
import TopRiskyEntitiesByVulnerabilities from '../widgets/TopRiskyEntitiesByVulnerabilities';
import TopRiskiestEntities from '../widgets/TopRiskiestEntities';
import RecentlyDetectedImageVulnerabilities from '../widgets/RecentlyDetectedImageVulnerabilities';
import MostCommonVulnerabilities from '../widgets/MostCommonVulnerabilities';
import ClustersWithMostClusterVulnerabilities from '../widgets/ClustersWithMostClusterVulnerabilities';
import CvesMenu from './CvesMenu';

const entityMenuTypes = [
    entityTypes.CLUSTER,
    entityTypes.NAMESPACE,
    entityTypes.DEPLOYMENT,
    entityTypes.NODE_COMPONENT,
    entityTypes.IMAGE_COMPONENT,
];

const VulnMgmtDashboardPage = () => {
    const navigate = useNavigate();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const workflowState = useContext(workflowStateContext);
    const searchState = workflowState.getCurrentSearchState();

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

        navigate(targetUrl);
    }

    const cveFilter = searchState.Fixable ? 'Fixable' : 'All';

    const headerComponents = (
        <>
            <PageTitle title="Vulnerability Management - Dashboard" />
            <div className="flex items-center">
                <div className="flex h-full mr-3 pr-3 border-r-2 border-base-400">
                    <div
                        className="flex mr-2"
                        style={{
                            backgroundColor: 'var(--pf-v5-global--palette--red-50)',
                        }}
                    >
                        <CvesMenu />
                    </div>
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
            </div>
        </>
    );
    return (
        <>
            <DashboardLayout
                banner={hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
                headerText="Vulnerability Management"
                headerComponents={headerComponents}
            >
                <div className="s-2 md:sx-4 xxxl:sx-4 ">
                    <TopRiskyEntitiesByVulnerabilities
                        defaultSelection={entityTypes.DEPLOYMENT}
                        cveFilter={cveFilter}
                    />
                </div>
                <div className="s-2 xxxl:sx-2">
                    <TopRiskiestEntities search={searchState} limit={DASHBOARD_LIMIT} />
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
                    <ClustersWithMostClusterVulnerabilities />
                </div>
            </DashboardLayout>
        </>
    );
};

export default VulnMgmtDashboardPage;

import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { useTheme } from 'Containers/ThemeProvider';
import PageHeader from 'Components/PageHeader';
import SearchFilterInput from 'Components/SearchFilterInput';
import DashboardCompliance from 'Containers/Dashboard/DashboardCompliance';
import TopRiskyDeployments from 'Containers/Dashboard/TopRiskyDeployments';
import useURLSearch from 'hooks/useURLSearch';
import {
    fetchAlertsByTimeseries,
    fetchSummaryAlertCountsLegacy as fetchSummaryAlertCounts,
} from 'services/AlertsService';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { fetchDeployments } from 'services/DeploymentsService';
import AlertsByTimeseriesChart from './AlertsByTimeseriesChart';
import SummaryCounts from './SummaryCounts';
import ViolationsByClusterChart from './ViolationsByClusterChart';
import ViolationsByPolicyCategory from './ViolationsByPolicyCategory';
import EnvironmentRisk from './EnvironmentRisk';

// The search filter value could either be `undefined`, a string, or a string[], so
// we coerce to an array for consistency.
function getFilteredClusters(clusterSearchValue) {
    if (Array.isArray(clusterSearchValue)) {
        return clusterSearchValue;
    }
    if (clusterSearchValue) {
        return [clusterSearchValue];
    }
    return [];
}

const DashboardPage = () => {
    const { isDarkMode } = useTheme();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const isViewFiltered =
        Object.keys(searchFilter).length &&
        Object.values(searchFilter).some((filter) => filter !== '');
    const subHeader = isViewFiltered ? 'Filtered view' : 'Default view';

    const [globalViolationsCounts, setGlobalViolationsCounts] = useState([]);
    const [violationCountsByPolicyCategories, setViolationCountsByPolicyCategories] = useState([]);
    const [alertsByTimeseries, setAlertsByTimeseries] = useState([]);
    const [topRiskyDeployments, setTopRiskyDeployments] = useState([]);

    const filteredClusters = getFilteredClusters(searchFilter.Cluster);

    // TODO All of these effects need cancellation/cleanup
    useEffect(() => {
        const query = getRequestQueryStringForSearchFilter(searchFilter);

        fetchSummaryAlertCounts({ 'request.query': query, group_by: 'CLUSTER' })
            .then(setGlobalViolationsCounts)
            .catch(() => {
                // TODO
            });

        fetchSummaryAlertCounts({ 'request.query': query, group_by: 'CATEGORY' })
            .then(setViolationCountsByPolicyCategories)
            .catch(() => {
                // TODO
            });

        fetchAlertsByTimeseries({ query })
            .then(setAlertsByTimeseries)
            .catch(() => {
                // TODO
            });

        const legacyFormatSearchOptions = Object.entries(searchFilter)
            .filter(([, value]) => value)
            .flatMap(([label, value]) => [
                { value: `${label}:`, type: 'categoryOption' },
                { value },
            ]);
        const sortOption = { field: 'Deployment Risk Priority', reversed: false };
        // Fetch the top 5 deployments, sorted by Risk
        fetchDeployments(legacyFormatSearchOptions, sortOption, 0, 5)
            .then((deployments) => {
                setTopRiskyDeployments(deployments.map(({ deployment }) => deployment));
            })
            .catch(() => {
                // TODO
            });
    }, [searchFilter]);

    return (
        <section
            className={`flex flex-1 h-full w-full ${!isDarkMode ? 'bg-base-200' : 'bg-base-0'}`}
        >
            <div className="flex flex-col w-full">
                <SummaryCounts />
                <div>
                    <PageHeader header="Dashboard" subHeader={subHeader}>
                        <SearchFilterInput
                            className="pf-u-w-100"
                            handleChangeSearchFilter={setSearchFilter}
                            placeholder="Add one or more resource filters"
                            searchFilter={searchFilter}
                            searchOptions={['Cluster']}
                        />
                    </PageHeader>
                </div>
                <div className="overflow-auto z-0">
                    <div
                        className={`flex flex-wrap ${!isDarkMode ? 'bg-base-300' : 'bg-base-100'}`}
                    >
                        <div className="w-full lg:w-1/2 p-6 z-1">
                            <EnvironmentRisk
                                globalViolationsCounts={globalViolationsCounts}
                                clusters={filteredClusters}
                            />
                        </div>
                        <div className="w-full lg:w-1/2 p-6 z-1 border-l border-base-400">
                            <DashboardCompliance />
                        </div>
                    </div>
                    <div className="overflow-auto relative border-t border-base-400">
                        <div className="flex flex-col w-full items-center overflow-hidden">
                            <div className="flex w-full flex-wrap -mx-6 p-3">
                                <div className="w-full lg:w-1/2 xl:w-1/3 p-3">
                                    <div className="flex flex-col bg-base-100 rounded-sm shadow h-full rounded">
                                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-base-300 border-b">
                                            <Icon.Layers className="h-4 w-4 m-3" />
                                            <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                                                Violations by Cluster
                                            </span>
                                        </h2>
                                        <div className="m-4 h-64">
                                            <ViolationsByClusterChart
                                                globalViolationsCounts={globalViolationsCounts}
                                            />
                                        </div>
                                    </div>
                                </div>
                                <div className="p-3 w-full lg:w-1/2 xl:w-1/3">
                                    <TopRiskyDeployments deployments={topRiskyDeployments} />
                                </div>
                                <div className="p-3 w-full lg:w-1/2 xl:w-1/3">
                                    <div className="flex flex-col bg-base-100 rounded-sm shadow h-full rounded">
                                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-base-300 border-b">
                                            <Icon.AlertTriangle className="h-4 w-4 m-3" />
                                            <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                                                Active Violations by Time
                                            </span>
                                        </h2>
                                        <div className="m-4 h-64">
                                            <AlertsByTimeseriesChart
                                                alertsByTimeseries={alertsByTimeseries}
                                            />
                                        </div>
                                    </div>
                                </div>
                                <ViolationsByPolicyCategory
                                    data={violationCountsByPolicyCategories}
                                    clusters={filteredClusters}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    );
};

DashboardPage.propTypes = {
    history: PropTypes.shape({
        push: PropTypes.func.isRequired,
    }).isRequired,
};

export default DashboardPage;

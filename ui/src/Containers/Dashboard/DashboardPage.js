import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import DashboardCompliance from 'Containers/Dashboard/DashboardCompliance';
import TopRiskyDeployments from 'Containers/Dashboard/TopRiskyDeployments';
import { selectors } from 'reducers';
import { actions as dashboardActions } from 'reducers/dashboard';
import AlertsByTimeseriesChart from './AlertsByTimeseriesChart';
import ViolationsByClusterChart from './ViolationsByClusterChart';
import ViolationsByPolicyCategory from './ViolationsByPolicyCategory';
import EnvironmentRisk from './EnvironmentRisk';

export const severityPropType = PropTypes.oneOf([
    'CRITICAL_SEVERITY',
    'HIGH_SEVERITY',
    'MEDIUM_SEVERITY',
    'LOW_SEVERITY'
]);

const DashboardPage = props => {
    const subHeader = props.isViewFiltered ? 'Filtered view' : 'Default view';
    return (
        <section className="flex flex-1 h-full w-full bg-base-200">
            <div className="flex flex-col w-full">
                <div>
                    <PageHeader header="Dashboard" subHeader={subHeader}>
                        <SearchInput
                            className="w-full"
                            searchOptions={props.searchOptions}
                            searchModifiers={props.searchModifiers}
                            searchSuggestions={props.searchSuggestions}
                            setSearchOptions={props.setSearchOptions}
                            setSearchModifiers={props.setSearchModifiers}
                            setSearchSuggestions={props.setSearchSuggestions}
                        />
                    </PageHeader>
                </div>
                <div className="overflow-auto bg-base-200 z-0">
                    <div className="flex flex-wrap bg-base-300 bg-dashboard">
                        <div className="w-full lg:w-1/2 p-6 z-1">
                            <EnvironmentRisk />
                        </div>
                        <div className="w-full lg:w-1/2 p-6 z-1 border-l border-base-400">
                            <DashboardCompliance />
                        </div>
                    </div>
                    <div className="overflow-auto bg-base-200 relative border-t border-base-400">
                        <div className="flex flex-col w-full items-center overflow-hidden">
                            <div className="flex w-full flex-wrap -mx-6 p-3">
                                <div className="w-full lg:w-1/2 xl:w-1/3 p-3">
                                    <div className="flex flex-col bg-base-100 rounded-sm shadow h-full rounded">
                                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-primary-200 border-b">
                                            <Icon.Layers className="h-4 w-4 m-3" />
                                            <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                                                Violations by Cluster
                                            </span>
                                        </h2>
                                        <div className="m-4 h-64">
                                            <ViolationsByClusterChart />
                                        </div>
                                    </div>
                                </div>
                                <div className="p-3 w-full lg:w-1/2 xl:w-1/3">
                                    <div className="flex flex-col bg-base-100 rounded-sm shadow h-full rounded">
                                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-primary-200 border-b">
                                            <Icon.AlertTriangle className="h-4 w-4 m-3" />
                                            <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                                                Active Violations by Time
                                            </span>
                                        </h2>
                                        <div className="m-4 h-64">
                                            <AlertsByTimeseriesChart />
                                        </div>
                                    </div>
                                </div>
                                <ViolationsByPolicyCategory />
                                <div className="p-3 w-full lg:w-1/2 xl:w-1/3">
                                    <TopRiskyDeployments />
                                </div>
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
        push: PropTypes.func.isRequired
    }).isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
    searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
    setSearchOptions: PropTypes.func.isRequired,
    setSearchModifiers: PropTypes.func.isRequired,
    setSearchSuggestions: PropTypes.func.isRequired,
    isViewFiltered: PropTypes.bool.isRequired
};

const isViewFiltered = createSelector(
    [selectors.getDashboardSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getDashboardSearchOptions,
    searchModifiers: selectors.getDashboardSearchModifiers,
    searchSuggestions: selectors.getDashboardSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = {
    setSearchOptions: dashboardActions.setDashboardSearchOptions,
    setSearchModifiers: dashboardActions.setDashboardSearchModifiers,
    setSearchSuggestions: dashboardActions.setDashboardSearchSuggestions
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(DashboardPage);

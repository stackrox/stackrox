import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import NoResultsMessage from 'Components/NoResultsMessage';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import TwoLevelPieChart from 'Components/visuals/TwoLevelPieChart';
import DashboardCompliance from 'Containers/Dashboard/DashboardCompliance';
import SeverityTile from 'Containers/Dashboard/SeverityTile';
import TopRiskyDeployments from 'Containers/Dashboard/TopRiskyDeployments';
import { severityLabels } from 'messages/common';
import { selectors } from 'reducers';
import { actions as dashboardActions } from 'reducers/dashboard';
import severityColorMap from 'constants/severityColors';
import AlertsByTimeseriesChart from 'Containers/Dashboard/AlertsByTimeseriesChart';
import ViolationsByClusterChart from './ViolationsByClusterChart';

const severityPropType = PropTypes.oneOf([
    'CRITICAL_SEVERITY',
    'HIGH_SEVERITY',
    'MEDIUM_SEVERITY',
    'LOW_SEVERITY'
]);

const groupedViolationsPropType = PropTypes.arrayOf(
    PropTypes.shape({
        counts: PropTypes.arrayOf(
            PropTypes.shape({
                count: PropTypes.string.isRequired,
                severity: severityPropType
            })
        ),
        group: PropTypes.string.isRequired
    })
);

class DashboardPage extends Component {
    static propTypes = {
        violatonsByPolicyCategory: groupedViolationsPropType.isRequired,
        globalViolationsCounts: groupedViolationsPropType.isRequired,
        violationsByCluster: groupedViolationsPropType.isRequired,
        alertsByTimeseries: PropTypes.arrayOf(PropTypes.shape()).isRequired,
        deployments: PropTypes.arrayOf(PropTypes.object).isRequired,
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

    renderAlertsByTimeseries = () => {
        if (!this.props.alertsByTimeseries || !this.props.alertsByTimeseries.length) {
            return <NoResultsMessage />;
        }
        return <AlertsByTimeseriesChart clusterData={this.props.alertsByTimeseries} />;
    };

    renderViolationsByCluster = () => {
        if (!this.props.violationsByCluster || !this.props.violationsByCluster.length) {
            return <NoResultsMessage />;
        }
        const clusterCharts = [];

        let i = 0;
        const limit = 4;
        while (i < this.props.violationsByCluster.length) {
            let j = i;
            let groupIndex = 0;
            const barCharts = [];
            while (j < this.props.violationsByCluster.length && groupIndex < limit) {
                const cluster = this.props.violationsByCluster[j];
                const dataPoint = {
                    name: cluster.group,
                    Critical: 0,
                    High: 0,
                    Medium: 0,
                    Low: 0
                };
                cluster.counts.forEach(d => {
                    dataPoint[severityLabels[d.severity]] = parseInt(d.count, 10);
                });
                barCharts.push(dataPoint);
                j += 1;
                groupIndex += 1;
            }
            clusterCharts.push(barCharts);
            i += 4;
        }
        return <ViolationsByClusterChart clusterCharts={clusterCharts} />;
    };

    renderViolationsByPolicyCategory = () => {
        if (!this.props.violatonsByPolicyCategory) return '';
        return this.props.violatonsByPolicyCategory.map(policyType => {
            const data = policyType.counts.map(d => ({
                name: severityLabels[d.severity],
                value: parseInt(d.count, 10),
                color: severityColorMap[d.severity],
                onClick: () => {
                    this.props.history.push(
                        `/main/violations?category=${policyType.group}&severity=${d.severity}`
                    );
                }
            }));
            return (
                <div className="p-3 w-full lg:w-1/2 xl:w-1/3" key={policyType.group}>
                    <div className="bg-base-100 rounded-sm shadow h-full rounded">
                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-primary-200 border-b">
                            <Icon.BarChart className="h-4 w-4 m-3" />
                            <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                                {policyType.group}
                            </span>
                        </h2>
                        <div className="m-4 h-64">
                            <TwoLevelPieChart data={data} />
                        </div>
                    </div>
                </div>
            );
        });
    };

    renderEnvironmentRisk = () => {
        const counts = {
            CRITICAL_SEVERITY: 0,
            HIGH_SEVERITY: 0,
            MEDIUM_SEVERITY: 0,
            LOW_SEVERITY: 0
        };
        this.props.globalViolationsCounts.forEach(group => {
            group.counts.forEach(d => {
                const count = parseInt(d.count, 10);
                counts[d.severity] += count;
            });
        });
        const severities = Object.keys(counts);
        const totalViolations = Object.values(counts).reduce((a, b) => a + b);
        return (
            <div className="w-full">
                <h2 className="-ml-6 bg-base-100 inline-block leading-normal mb-6 p-3 pb-2 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase">
                    {totalViolations === 1
                        ? `${totalViolations} System Violation`
                        : `${totalViolations} System Violations`}
                </h2>
                <div className="flex">
                    {severities.map((severity, i) => (
                        <SeverityTile
                            severity={severity}
                            count={counts[severity]}
                            color={severityColorMap[severity]}
                            index={i}
                            key={severity}
                        />
                    ))}
                </div>
            </div>
        );
    };

    renderTopRiskyDeployments = () => {
        if (!this.props.deployments) return '';
        return <TopRiskyDeployments deployments={this.props.deployments} />;
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full w-full bg-base-200">
                <div className="flex flex-col w-full">
                    <div className="z-1">
                        <PageHeader header="Dashboard" subHeader={subHeader}>
                            <SearchInput
                                className="w-full"
                                searchOptions={this.props.searchOptions}
                                searchModifiers={this.props.searchModifiers}
                                searchSuggestions={this.props.searchSuggestions}
                                setSearchOptions={this.props.setSearchOptions}
                                setSearchModifiers={this.props.setSearchModifiers}
                                setSearchSuggestions={this.props.setSearchSuggestions}
                            />
                        </PageHeader>
                    </div>
                    <div className="overflow-auto bg-base-200 z-0">
                        <div className="flex flex-wrap bg-base-300 bg-dashboard">
                            <div className="w-full lg:w-1/2 p-6 z-1">
                                {this.renderEnvironmentRisk()}
                            </div>
                            <div className="w-full lg:w-1/2 py-6 border-l-2 border-base-400 z-1">
                                <div className="p-0 h-full w-full dashboard-benchmarks">
                                    <DashboardCompliance />
                                </div>
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
                                                {this.renderViolationsByCluster()}
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
                                                {this.renderAlertsByTimeseries()}
                                            </div>
                                        </div>
                                    </div>
                                    {this.renderViolationsByPolicyCategory()}
                                    <div className="p-3 w-full lg:w-1/2 xl:w-1/3">
                                        {this.renderTopRiskyDeployments()}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        );
    }
}

const getTopRiskyDeployments = createSelector(
    [selectors.getFilteredDeployments],
    deployments => deployments.sort((a, b) => a.priority - b.priority).slice(0, 5)
);

const isViewFiltered = createSelector(
    [selectors.getDashboardSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    violatonsByPolicyCategory: selectors.getAlertCountsByPolicyCategories,
    globalViolationsCounts: selectors.getGlobalAlertCounts,
    violationsByCluster: selectors.getAlertCountsByCluster,
    alertsByTimeseries: selectors.getAlertsByTimeseries,
    deployments: getTopRiskyDeployments,
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

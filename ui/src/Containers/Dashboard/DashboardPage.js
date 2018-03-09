import React, { Component } from 'react';
import PropTypes from 'prop-types';
import {
    Line,
    BarChart,
    Bar,
    Cell,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer
} from 'recharts';
import { format, subDays } from 'date-fns';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import TwoLevelPieChart from 'Components/visuals/TwoLevelPieChart';
import CustomLineChart from 'Components/visuals/CustomLineChart';
import DashboardBenchmarks from 'Containers/Dashboard/DashboardBenchmarks';
import SeverityTile from 'Containers/Dashboard/SeverityTile';
import { severityLabels } from 'messages/common';
import { selectors } from 'reducers';

//  @TODO: Have one source of truth for severity colors
const severityColorMap = {
    CRITICAL_SEVERITY: 'hsl(7, 100%, 55%)',
    HIGH_SEVERITY: 'hsl(349, 100%, 78%)',
    MEDIUM_SEVERITY: 'hsl(20, 100%, 78%)',
    LOW_SEVERITY: 'hsl(42, 100%, 84%)'
};

const policyCategoriesMap = {
    'Container Configuration': {
        icon: <Icon.Grid className="h-4 w-4 mr-3" />
    },
    'Image Assurance': {
        icon: <Icon.Copy className="h-4 w-4 mr-3" />
    },
    'Privileges and Capabilities': {
        icon: <Icon.Lock className="h-4 w-4 mr-3" />
    }
};

const benchmarkPropType = PropTypes.arrayOf(
    PropTypes.shape({
        checks: PropTypes.arrayOf(
            PropTypes.shape({
                aggregatedResults: PropTypes.shape({
                    PASS: PropTypes.number,
                    INFO: PropTypes.number,
                    WARN: PropTypes.number,
                    NOTE: PropTypes.number
                }),
                definition: PropTypes.shape({
                    description: PropTypes.string,
                    name: PropTypes.string
                }),
                hostResults: PropTypes.arrayOf(
                    PropTypes.shape({
                        host: PropTypes.string,
                        notes: PropTypes.arrayOf(PropTypes.string),
                        result: PropTypes.string
                    })
                )
            })
        )
    })
);

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
        violationsByCluster: groupedViolationsPropType.isRequired,
        alertsByTimeseries: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
                severity: severityPropType.isRequired,
                time: PropTypes.string.isRequired,
                type: PropTypes.string.isRequired
            })
        ).isRequired,
        benchmarks: PropTypes.shape({
            'CIS Docker v1.1.0 Benchmark': benchmarkPropType,
            'CIS Swarm v1.1.0 Benchmark': benchmarkPropType,
            'CIS Kubernetes v1.2.0 Benchmark': benchmarkPropType
        }).isRequired,
        clustersByName: PropTypes.object.isRequired, // eslint-disable-line react/forbid-prop-types
        history: PropTypes.shape({
            push: PropTypes.func.isRequired
        }).isRequired
    };

    makeBarClickHandler = (clusterName, severity) => () => {
        const cluster = this.props.clustersByName[clusterName];
        // if clusters are not loaded yet, at least we can redirect to unfiltered violations
        const clusterQuery = cluster ? `cluster=${cluster.id}` : '';
        this.props.history.push(`/main/violations?severity=${severity}&${clusterQuery}`);
    };

    renderAlertsByTimeseries = () => {
        if (!this.props.alertsByTimeseries) return '';
        const timeAlertMap = {};
        const xAxisBuckets = [];
        for (let i = 6; i >= 0; i -= 1) {
            const key = format(subDays(new Date(), i), 'MMM DD');
            timeAlertMap[key] = 0;
            xAxisBuckets.push(key);
        }
        let startCount = 0;
        this.props.alertsByTimeseries.forEach(alert => {
            const time = format(parseInt(alert.time, 10), 'MMM DD');
            const alerts = timeAlertMap[time];
            if (alerts !== undefined) {
                switch (alert.type) {
                    case 'CREATED':
                        timeAlertMap[time] += 1;
                        break;
                    case 'REMOVED':
                        timeAlertMap[time] -= 1;
                        break;
                    default:
                        break;
                }
            } else {
                switch (alert.type) {
                    case 'CREATED':
                        startCount += 1;
                        break;
                    case 'REMOVED':
                        startCount -= 1;
                        break;
                    default:
                        break;
                }
            }
        });
        let runningSum = startCount;
        xAxisBuckets.forEach(key => {
            const prevVal = timeAlertMap[key];
            timeAlertMap[key] += runningSum;
            runningSum += prevVal;
        });
        const data = Object.keys(timeAlertMap).map(time => ({
            time,
            violations: timeAlertMap[time]
        }));
        return (
            <CustomLineChart data={data} xAxisDataKey="time" yAxisDataKey="">
                <Line type="monotone" dataKey="violations" stroke="#82ca9d" />
            </CustomLineChart>
        );
    };

    renderViolationsByCluster = () => {
        if (!this.props.violationsByCluster) return '';
        const data = this.props.violationsByCluster.map(cluster => {
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
            return dataPoint;
        });
        return (
            <ResponsiveContainer>
                <BarChart
                    data={data}
                    margin={{
                        top: 5,
                        right: 30,
                        left: 20,
                        bottom: 5
                    }}
                >
                    <XAxis dataKey="name" />
                    <YAxis
                        domain={[0, 'dataMax']}
                        allowDecimals={false}
                        label={{
                            value: 'Count',
                            angle: -90,
                            position: 'insideLeft',
                            textAnchor: 'middle'
                        }}
                    />
                    <CartesianGrid strokeDasharray="3 3" />
                    <Tooltip />
                    <Legend horizontalAlign="right" wrapperStyle={{ lineHeight: '40px' }} />
                    {Object.keys(severityLabels).map(severity => {
                        const arr = [];
                        const bar = (
                            <Bar
                                name={severityLabels[severity]}
                                key={severityLabels[severity]}
                                dataKey={severityLabels[severity]}
                                fill={severityColorMap[severity]}
                            >
                                {data.map(entry => (
                                    <Cell
                                        key={entry.name}
                                        className="cursor-pointer"
                                        onClick={this.makeBarClickHandler(entry.name, severity)}
                                    />
                                ))}
                            </Bar>
                        );
                        arr.push(bar);
                        return arr;
                    })}
                </BarChart>
            </ResponsiveContainer>
        );
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
                <div className="p-8 w-full lg:w-1/2" key={policyType.group}>
                    <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                            {policyCategoriesMap[policyType.group].icon}
                            {policyType.group}
                        </h2>
                        <div className="flex flex-1 m-4 h-64">
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
        this.props.violationsByCluster.forEach(cluster => {
            cluster.counts.forEach(d => {
                const count = parseInt(d.count, 10);
                counts[d.severity] += count;
            });
        });
        const severities = Object.keys(counts);
        return (
            <div className="flex flex-1 flex-col w-full">
                <h2 className="flex items-center text-xl text-base font-sans text-base-600 pb-8 tracking-wide font-500">
                    Environment Risk
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

    renderBenchmarks = () => (
        <div className="flex flex-1 flex-col w-full">
            <h2 className="flex items-center text-xl text-base font-sans text-base-600 pb-8 tracking-wide font-500">
                Benchmarks
            </h2>
            <DashboardBenchmarks benchmarks={this.props.benchmarks} />
        </div>
    );

    render() {
        return (
            <section className="w-full h-full transition">
                <div className="flex bg-white border-b border-primary-500">
                    <div className="flex flex-1 flex-col w-1/2 p-8">
                        {this.renderEnvironmentRisk()}
                    </div>
                    <div className="flex flex-1 flex-col w-1/2 p-8 border-l border-primary-200">
                        {this.renderBenchmarks()}
                    </div>
                </div>
                <div className="overflow-auto">
                    <div className="flex flex-col w-full">
                        <div className="flex w-full flex-wrap">
                            <div className="p-8 md:w-full lg:w-1/2">
                                <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                                    <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                                        <Icon.Layers className="h-4 w-4 mr-3" />
                                        Violations by Cluster
                                    </h2>
                                    <div className="flex flex-1 m-4 h-64">
                                        {this.renderViolationsByCluster()}
                                    </div>
                                </div>
                            </div>
                            <div className="p-8 md:w-full lg:w-1/2">
                                <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                                    <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                                        <Icon.AlertTriangle className="h-4 w-4 mr-3" />
                                        Active Violations by Time
                                    </h2>
                                    <div className="flex flex-1 m-4 h-64">
                                        {this.renderAlertsByTimeseries()}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="flex flex-col w-full">
                        <div className="flex w-full flex-wrap">
                            {this.renderViolationsByPolicyCategory()}
                        </div>
                    </div>
                </div>
            </section>
        );
    }
}

const getClustersByName = createSelector([selectors.getClusters], clusters =>
    clusters.reduce(
        (result, cluster) => ({
            ...result,
            [cluster.name]: cluster
        }),
        {}
    )
);

const mapStateToProps = createStructuredSelector({
    violatonsByPolicyCategory: selectors.getAlertCountsByPolicyCategories,
    violationsByCluster: selectors.getAlertCountsByCluster,
    alertsByTimeseries: selectors.getAlertsByTimeseries,
    benchmarks: selectors.getBenchmarks,
    clustersByName: getClustersByName
});

export default connect(mapStateToProps)(DashboardPage);

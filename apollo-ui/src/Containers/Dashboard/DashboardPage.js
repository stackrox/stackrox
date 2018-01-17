import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';
import TwoLevelPieChart from 'Components/visuals/TwoLevelPieChart';

import axios from 'axios';

//  @TODO: Have one source of truth for severity colors
const severityColorMap = {
    CRITICAL_SEVERITY: 'hsl(7, 100%, 55%)',
    HIGH_SEVERITY: 'hsl(8, 87%, 67%)',
    MEDIUM_SEVERITY: 'hsl(38, 80%, 57%)',
    LOW_SEVERITY: 'hsl(54, 84%, 65%)'
};

const severityLabels = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low'
};

const policyCategoriesLabels = {
    CONTAINER_CONFIGURATION: 'Container Configuration',
    IMAGE_ASSURANCE: 'Image Assurance',
    PRIVILEGES_CAPABILITIES: 'Privileges and Capabilities'
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_VIOLATIONS_BY_POLICY_CATEGORY':
            return { violatonsByPolicyCategory: nextState.violatonsByPolicyCategory };
        case 'UPDATE_VIOLATIONS_BY_CLUSTERS':
            return { violationsByCluster: nextState.violationsByCluster };
        default:
            return prevState;
    }
};

class DashboardPage extends Component {
    static propTypes = {
        history: PropTypes.shape({
            push: PropTypes.func.isRequired
        }).isRequired
    }

    constructor(props) {
        super(props);

        this.colorBy = d => d.color;

        this.state = {
            violatonsByPolicyCategory: [],
            violationsByCluster: []
        };
    }

    componentDidMount() {
        this.getViolationsByPolicyCategory();
        this.getViolationsByCluster();
    }

    getViolationsByPolicyCategory = () => axios.get('/v1/alerts/counts', {
        params: { group_by: 'CATEGORY', 'request.stale': false }
    }).then((response) => {
        const violatonsByPolicyCategory = response.data.groups;
        if (!violatonsByPolicyCategory) return;
        this.update('UPDATE_VIOLATIONS_BY_POLICY_CATEGORY', { violatonsByPolicyCategory });
    }).catch((error) => {
        console.error(error);
    });

    getViolationsByCluster = () => axios.get('/v1/alerts/counts', {
        params: { group_by: 'CLUSTER', 'request.stale': false }
    }).then((response) => {
        const violationsByCluster = response.data.groups;
        if (!violationsByCluster) return;
        this.update('UPDATE_VIOLATIONS_BY_CLUSTERS', { violationsByCluster });
    }).catch((error) => {
        console.error(error);
    });

    makeBarClickHandler = (cluster, severity) => () => {
        this.props.history.push(`/violations?severity=${severity}&cluster=${cluster}`);
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderViolationsByCluster = () => {
        if (!this.state.violationsByCluster) return '';
        const data = this.state.violationsByCluster.map((cluster) => {
            const dataPoint = {
                name: cluster.group,
                Critical: 0,
                High: 0,
                Medium: 0,
                Low: 0
            };
            cluster.counts.forEach((d) => {
                dataPoint[severityLabels[d.severity]] = parseInt(d.count, 10);
            });
            return dataPoint;
        });
        return (
            <BarChart
                width={600}
                height={300}
                data={data}
                margin={
                    {
                        top: 5,
                        right: 30,
                        left: 20,
                        bottom: 5
                    }
                }
            >
                <XAxis dataKey="name" />
                <YAxis
                    domain={[0, 'dataMax']}
                    allowDecimals={false}
                    label={{
                        value: 'Count', angle: -90, position: 'insideLeft', textAnchor: 'middle'
                    }}
                />
                <CartesianGrid strokeDasharray="3 3" />
                <Tooltip />
                <Legend horizontalAlign="right" wrapperStyle={{ lineHeight: '40px' }} />
                {
                    Object.keys(severityLabels).map(severity => (
                        <Bar name={severityLabels[severity]} key={severityLabels[severity]} dataKey={severityLabels[severity]} fill={severityColorMap[severity]}>
                            {data.map(entry =>
                                <Cell key={entry.name} className="cursor-pointer" onClick={this.makeBarClickHandler(entry.name, severity)} />)}
                        </Bar>
                    ))
                }
            </BarChart>
        );
    }

    renderViolationsByPolicyCategory = () => {
        if (!this.state.violatonsByPolicyCategory) return '';
        return this.state.violatonsByPolicyCategory.map((policyType) => {
            const data = policyType.counts.map(d => ({
                name: severityLabels[d.severity],
                value: parseInt(d.count, 10),
                color: severityColorMap[d.severity],
                onClick: () => {
                    this.props.history.push(`/violations?category=${policyType.group}&severity=${d.severity}`);
                }
            }));
            return (
                <div className="md:w-1/2 sm:w-full" key={policyType.group}>
                    <div className="m-4 p-2 bg-white rounded-sm shadow">
                        <div className="capitalize">{policyCategoriesLabels[policyType.group]}</div>
                        <div className="h-64">
                            <TwoLevelPieChart data={data} />
                        </div>
                    </div>
                </div>
            );
        });
    };

    render() {
        return (
            <section className="flex flex-col w-full h-full">
                <h1 className="font-500 mx-3 border-b border-primary-300 py-4 uppercase text-xl font-800 text-primary-600 tracking-wide">Environment Risk</h1>
                <div className="overflow-auto">
                    <div className="flex flex-col w-full">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">Violations by Cluster</h2>
                        <div className="flex flex-1 w-full flex-wrap">
                            <div className="m-4 p-4 bg-white rounded-sm shadow">
                                {this.renderViolationsByCluster()}
                            </div>
                        </div>
                    </div>
                    <div className="flex flex-col w-full">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">Violations by Policy Category</h2>
                        <div className="flex flex-1 w-full flex-wrap">
                            {this.renderViolationsByPolicyCategory()}
                        </div>
                    </div>
                </div>
            </section>
        );
    }
}

export default DashboardPage;

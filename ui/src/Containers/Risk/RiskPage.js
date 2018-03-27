import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import Collapsible from 'react-collapsible';

import { selectors } from 'reducers';

import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import KeyValuePairs from 'Components/KeyValuePairs';

const deploymentDetailsMap = {
    id: {
        label: 'Deployment ID'
    },
    clusterName: {
        label: 'Cluster'
    },
    namespace: {
        label: 'Namespace'
    },
    replicas: {
        label: 'Replicas'
    },
    labels: {
        label: 'Labels'
    },
    ports: {
        label: 'Port configuration'
    },
    mounts: {
        label: 'Mounts'
    },
    volume: {
        label: 'Volume'
    }
};

const containerConfigMap = {
    args: {
        label: 'Args'
    },
    command: {
        label: 'Command'
    },
    directory: {
        label: 'Directory'
    },
    env: {
        label: 'Environment'
    },
    uid: {
        label: 'User ID'
    },
    user: {
        label: 'User'
    }
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'SELECT_DEPLOYMENT':
            return { selectedDeployment: nextState.deployment };
        case 'UNSELECT_DEPLOYMENT':
            return { selectedDeployment: null };
        default:
            return prevState;
    }
};

class RiskPage extends Component {
    static propTypes = {
        deployments: PropTypes.arrayOf(PropTypes.object).isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            selectedDeployment: null
        };
    }

    onDeploymentClick = deployment => {
        this.selectDeployment(deployment);
    };

    selectDeployment = deployment => {
        this.update('SELECT_DEPLOYMENT', { deployment });
    };

    unselectDeployment = () => {
        this.update('UNSELECT_DEPLOYMENT');
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderTable() {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'clusterName', label: 'Cluster' },
            { key: 'namespace', label: 'Namespace' },
            { key: 'risk.score', label: 'Priority' }
        ];
        const rows = this.props.deployments;
        return <Table columns={columns} rows={rows} onRowClick={this.onDeploymentClick} />;
    }

    renderCollapsibleCard = (title, direction) => {
        const icons = {
            up: <Icon.ChevronUp className="h-4 w-4" />,
            down: <Icon.ChevronDown className="h-4 w-4" />
        };

        return (
            <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide cursor-pointer flex justify-between">
                <div>{title}</div>
                <div>{icons[direction]}</div>
            </div>
        );
    };

    renderSidePanel = () => {
        if (!this.state.selectedDeployment) return '';
        const header = this.state.selectedDeployment.name;
        const riskPanelTabs = [{ text: 'risk indicators' }, { text: 'deployment details' }];
        return (
            <Panel header={header} onClose={this.unselectDeployment} width="w-2/3">
                <Tabs headers={riskPanelTabs}>
                    {riskPanelTabs.map(tab => (
                        <TabContent key={tab.text}>{this.renderTab(tab.text)}</TabContent>
                    ))}
                </Tabs>
            </Panel>
        );
    };

    renderTab = tabText => {
        switch (tabText) {
            case 'risk indicators':
                return (
                    <div className="flex flex-1 flex-col bg-base-100">
                        {this.renderRiskIndicators()}
                    </div>
                );
            case 'deployment details':
                return (
                    <div className="flex flex-1 flex-col bg-base-100">
                        {this.renderOverview()}
                        {this.renderContainerConfig()}
                    </div>
                );
            default:
                return '';
        }
    };

    renderRiskIndicators = () => {
        if (!this.state.selectedDeployment.risk) return '';
        const { risk } = this.state.selectedDeployment;
        return (
            <div className="px-3 py-4">
                {risk.results.map(result => (
                    <div
                        className="alert-preview bg-white shadow text-primary-600 tracking-wide"
                        key={result.name}
                    >
                        <Collapsible
                            open
                            trigger={this.renderCollapsibleCard(result.name, 'up')}
                            triggerWhenOpen={this.renderCollapsibleCard(result.name, 'down')}
                            transitionTime={100}
                        >
                            {result.factors.map(factor => (
                                <div className="h-full p-3 font-500" key={factor}>
                                    <Icon.Circle className="h-2 w-2 mr-3" />
                                    {factor}
                                </div>
                            ))}
                        </Collapsible>
                    </div>
                ))}
            </div>
        );
    };

    renderOverview = () => {
        const title = 'Overview';
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full p-3">
                            <KeyValuePairs
                                data={this.state.selectedDeployment}
                                keyValueMap={deploymentDetailsMap}
                            />
                        </div>
                    </Collapsible>
                </div>
            </div>
        );
    };

    renderContainerConfig = () => {
        const title = 'Container configuration';
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full p-3">
                            {this.state.selectedDeployment.containers.map((container, index) => (
                                <KeyValuePairs
                                    data={container.config}
                                    keyValueMap={containerConfigMap}
                                    key={index}
                                />
                            ))}
                        </div>
                    </Collapsible>
                </div>
            </div>
        );
    };

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 self-end justify-end" />
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow border-t border-primary-300 bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    deployments: selectors.getDeployments
});

export default connect(mapStateToProps)(RiskPage);

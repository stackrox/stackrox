import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import Collapsible from 'react-collapsible';

import { selectors } from 'reducers';
import { actions as riskActions } from 'reducers/risk';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import KeyValuePairs from 'Components/KeyValuePairs';
import { Link } from 'react-router-dom';
import { sortNumber } from 'sorters/sorters';
import lowerCase from 'lodash/lowerCase';
import capitalize from 'lodash/capitalize';

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

class RiskPage extends Component {
    static propTypes = {
        deployments: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.match.isRequired
    };

    getSelectedDeployment = () => {
        if (this.props.match.params.id) {
            return this.props.deployments.find(
                deployment => deployment.id === this.props.match.params.id
            );
        }
        return null;
    };

    updateSelectedDeployment = deployment => {
        const urlSuffix = deployment && deployment.id ? `/${deployment.id}` : '';
        this.props.history.push({
            pathname: `/main/risk${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'clusterName', label: 'Cluster' },
            { key: 'namespace', label: 'Namespace' },
            { key: 'priority', label: 'Priority', sortMethod: sortNumber('priority') }
        ];
        const rows = this.props.deployments;
        return <Table columns={columns} rows={rows} onRowClick={this.updateSelectedDeployment} />;
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
        const selectedDeployment = this.getSelectedDeployment();
        if (!selectedDeployment) return null;
        const header = selectedDeployment.name;
        const riskPanelTabs = [{ text: 'risk indicators' }, { text: 'deployment details' }];
        return (
            <div className="w-1/2">
                <Panel header={header} onClose={this.updateSelectedDeployment}>
                    <Tabs headers={riskPanelTabs}>
                        {riskPanelTabs.map(tab => (
                            <TabContent key={tab.text}>{this.renderTab(tab.text)}</TabContent>
                        ))}
                    </Tabs>
                </Panel>
            </div>
        );
    };

    renderTab = tabText => {
        switch (tabText) {
            case 'risk indicators':
                return <div className="flex flex-1 flex-col">{this.renderRiskIndicators()}</div>;
            case 'deployment details':
                return (
                    <div className="flex flex-1 flex-col">
                        {this.renderOverview()}
                        {this.renderContainerConfig()}
                    </div>
                );
            default:
                return '';
        }
    };

    renderRiskIndicators = () => {
        const selectedDeployment = this.getSelectedDeployment();
        if (!selectedDeployment || !selectedDeployment.risk) return null;
        const { risk } = selectedDeployment;
        return risk.results.map(result => (
            <div className="px-3 py-4" key={result.name}>
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
                            <div className="flex h-full p-3 font-500" key={factor}>
                                <div>
                                    <Icon.Circle className="h-2 w-2 mr-3" />
                                </div>
                                <div className="pl-1">{factor}</div>
                            </div>
                        ))}
                    </Collapsible>
                </div>
            </div>
        ));
    };

    renderOverview = () => {
        const selectedDeployment = this.getSelectedDeployment();
        const title = 'Overview';
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide border border-base-200">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full p-3">
                            <KeyValuePairs
                                data={selectedDeployment}
                                keyValueMap={deploymentDetailsMap}
                            />
                        </div>
                    </Collapsible>
                </div>
            </div>
        );
    };

    renderContainerConfig = () => {
        const selectedDeployment = this.getSelectedDeployment();
        const title = 'Container configuration';
        return (
            <div className="px-3 py-4">
                <div className="alert-preview bg-white shadow text-primary-600 tracking-wide border border-base-200">
                    <Collapsible
                        open
                        trigger={this.renderCollapsibleCard(title, 'up')}
                        triggerWhenOpen={this.renderCollapsibleCard(title, 'down')}
                        transitionTime={100}
                    >
                        <div className="h-full p-3">
                            {selectedDeployment.containers.map((container, index) => {
                                if (!container.config) return null;
                                return (
                                    <div key={index}>
                                        <KeyValuePairs
                                            data={container.config}
                                            keyValueMap={containerConfigMap}
                                        />
                                        <div className="flex py-3">
                                            <div className="pr-1">Mounts:</div>
                                            <div className="-ml-8 mt-4 w-full">
                                                {container.volumes &&
                                                    container.volumes.length &&
                                                    container.volumes.map((volume, idx) => (
                                                        <div
                                                            key={idx}
                                                            className={`py-2 ${
                                                                idx === container.volumes.length - 1
                                                                    ? ''
                                                                    : 'border-base-300 border-b'
                                                            }`}
                                                        >
                                                            {Object.keys(volume).map(
                                                                (key, id) =>
                                                                    volume[key] !== '' && (
                                                                        <div
                                                                            key={`${
                                                                                volume.name
                                                                            }-${id}`}
                                                                            className="py-1 font-500"
                                                                        >
                                                                            <span className=" pr-1">
                                                                                {capitalize(
                                                                                    lowerCase(key)
                                                                                )}:
                                                                            </span>
                                                                            <span className="text-accent-400 italic">
                                                                                {volume[
                                                                                    key
                                                                                ].toString()}
                                                                            </span>
                                                                        </div>
                                                                    )
                                                            )}
                                                        </div>
                                                    ))}
                                            </div>
                                        </div>
                                        {container.image &&
                                            container.image.name &&
                                            container.image.name.fullName && (
                                                <div className="flex py-3">
                                                    <div className="pr-1">Image Name:</div>
                                                    <Link
                                                        className="font-500 text-primary-600 hover:text-primary-800"
                                                        to={`/main/images/${
                                                            container.image.name.sha
                                                        }`}
                                                    >
                                                        {container.image.name.fullName}
                                                    </Link>
                                                </div>
                                            )}
                                    </div>
                                );
                            })}
                        </div>
                    </Collapsible>
                </div>
            </div>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Risk" subHeader={subHeader}>
                        <SearchInput
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow bg-base-100 flex flex-1">
                            {this.renderTable()}
                        </div>
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getDeploymentsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    deployments: selectors.getDeployments,
    searchOptions: selectors.getDeploymentsSearchOptions,
    searchModifiers: selectors.getDeploymentsSearchModifiers,
    searchSuggestions: selectors.getDeploymentsSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = dispatch => ({
    setSearchOptions: searchOptions =>
        dispatch(riskActions.setDeploymentsSearchOptions(searchOptions)),
    setSearchModifiers: searchModifiers =>
        dispatch(riskActions.setDeploymentsSearchModifiers(searchModifiers)),
    setSearchSuggestions: searchSuggestions =>
        dispatch(riskActions.setDeploymentsSearchSuggestions(searchSuggestions))
});

export default connect(mapStateToProps, mapDispatchToProps)(RiskPage);

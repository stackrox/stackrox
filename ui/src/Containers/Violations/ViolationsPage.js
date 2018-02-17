import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import MultiSelect from 'react-select';
import axios from 'axios';
import queryString from 'query-string';
import isEqual from 'lodash/isEqual';

import { sortNumber, sortSeverity } from 'sorters/sorters';

import Table from 'Components/Table';
import PolicyAlertsSidePanel from './PolicyAlertsSidePanel';
import ViolationsModal from './ViolationsModal';

const policyCategoriesLabels = {
    CONTAINER_CONFIGURATION: 'Container Configuration',
    IMAGE_ASSURANCE: 'Image Assurance',
    PRIVILEGES_CAPABILITIES: 'Privileges and Capabilities'
};

const severityLabels = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low'
};

const setSeverityClass = item => {
    switch (item) {
        case 'Low':
            return 'text-low-500';
        case 'Medium':
            return 'text-medium-500';
        case 'High':
            return 'text-high-500';
        case 'Critical':
            return 'text-critical-500';
        default:
            return '';
    }
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_ALERTS_BY_POLICIES':
            return { alertsByPolicies: nextState.alertsByPolicies };
        case 'OPEN_POLICY_ALERTS_PANEL':
            return { isPanelOpen: true, policy: nextState.policy };
        case 'CLOSE_POLICY_ALERTS_PANEL':
            return { isPanelOpen: false, policy: null };
        case 'UPDATE_ALERT':
            return { alertId: nextState.alertId };
        default:
            return prevState;
    }
};

class ViolationsPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.match.isRequired
    };

    constructor(props) {
        super(props);

        this.pollTimeoutId = null;

        this.state = {
            category: {
                options: [
                    { label: 'Image Assurance', value: 'IMAGE_ASSURANCE' },
                    { label: 'Container Configuration', value: 'CONTAINER_CONFIGURATION' },
                    { label: 'Privileges & Capabilities', value: 'PRIVILEGES_CAPABILITIES' }
                ]
            },
            severity: {
                options: [
                    { label: 'Critical Severity', value: 'CRITICAL_SEVERITY' },
                    { label: 'High Severity', value: 'HIGH_SEVERITY' },
                    { label: 'Medium Severity', value: 'MEDIUM_SEVERITY' },
                    { label: 'Low Severity', value: 'LOW_SEVERITY' }
                ]
            },
            alertsByPolicies: [],
            policy: {},
            alertId: null
        };
    }

    componentDidMount() {
        this.pollAlertGroups();
        this.getAlertId();
    }

    componentWillUnmount() {
        if (this.pollTimeoutId) {
            clearTimeout(this.pollTimeoutId);
            this.pollTimeoutId = null;
        }
    }

    onActivePillsChange(active) {
        const { params } = this;
        params.category = Object.keys(active);
        this.getAlertsGroups();
    }

    onFilterChange = type => options => {
        this.props.history.push({
            pathname: this.props.location.pathname,
            search: queryString.stringify({
                ...this.getFilterParams(),
                [type]: options.map(c => c.value)
            })
        });

        // history will be updated asynchronously and it should happen before alerts fetching
        // TODO-ivan: to be removed with switching to react-router-redux
        setTimeout(this.getAlertsGroups, 0);
    };

    onModalClose = () => {
        this.changeUrl(alert.id);
        this.update('UPDATE_ALERT', { alertId: null });
    };

    getAlertsGroups = () => {
        const params = queryString.stringify({
            ...this.getFilterParams(),
            stale: false
        });
        return axios
            .get(`/v1/alerts/groups?${params}`)
            .then(response => {
                if (
                    !response.data.alertsByPolicies ||
                    isEqual(response.data.alertsByPolicies, this.state.alertsByPolicies)
                )
                    return;
                this.setState({ alertsByPolicies: response.data.alertsByPolicies });
            })
            .catch(error => {
                console.error(error);
            });
    };

    getFilterParams() {
        const { search } = this.props.location;
        const params = queryString.parse(search);
        return params;
    }

    getAlertId() {
        const { alertId } = this.props.match.params;
        this.update('UPDATE_ALERT', { alertId });
    }

    pollAlertGroups = () => {
        this.getAlertsGroups().then(() => {
            this.pollTimeoutId = setTimeout(this.pollAlertGroups, 5000);
        });
    };

    openPanel = policy => {
        this.update('OPEN_POLICY_ALERTS_PANEL', { policy });
    };

    closePanel = () => {
        this.update('CLOSE_POLICY_ALERTS_PANEL');
    };

    launchModal = alert => {
        this.changeUrl(alert.id);
        this.update('UPDATE_ALERT', { alertId: alert.id });
    };

    changeUrl(id) {
        const urlValue = id ? `/${id}` : '';
        this.props.history.push({
            pathname: `/main/violations${urlValue}`
        });
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderTable = () => {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'description', label: 'Description' },
            {
                key: 'categories',
                label: 'Categories',
                keyValueFunc: obj => (obj.length > 1 ? 'Multiple' : policyCategoriesLabels[obj[0]]),
                tooltip: categories =>
                    categories.map(category => policyCategoriesLabels[category]).join(' | ')
            },
            {
                key: 'severity',
                label: 'Severity',
                classFunc: setSeverityClass,
                sortMethod: sortSeverity
            },
            {
                key: 'numAlerts',
                label: 'Alerts',
                align: 'right',
                sortMethod: sortNumber('numAlerts')
            }
        ];
        const rows = this.state.alertsByPolicies.map(obj => {
            const row = {
                id: obj.policy.id,
                name: obj.policy.name,
                description: obj.policy.description,
                categories: obj.policy.categories,
                severity: severityLabels[obj.policy.severity],
                numAlerts: obj.numAlerts
            };
            return row;
        });
        return <Table columns={columns} rows={rows} onRowClick={this.openPanel} />;
    };

    renderModal() {
        if (!this.state.alertId) return '';
        return <ViolationsModal alertId={this.state.alertId} onClose={this.onModalClose} />;
    }

    renderPolicyAlertsPanel = () => {
        if (!this.state.isPanelOpen) return '';
        return (
            <PolicyAlertsSidePanel
                policy={this.state.policy}
                onClose={this.closePanel}
                onRowClick={this.launchModal}
                updatePolicy={this.updatePolicy}
            />
        );
    };

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 self-end justify-end">
                        <div className="flex ml-3">
                            <MultiSelect
                                multi
                                onChange={this.onFilterChange('category')}
                                options={this.state.category.options}
                                placeholder="Select categories"
                                removeSelected
                                value={this.getFilterParams().category}
                                className="text-base-600 font-400 min-w-64"
                            />
                        </div>
                        <div className="flex ml-3">
                            <MultiSelect
                                multi
                                onChange={this.onFilterChange('severity')}
                                options={this.state.severity.options}
                                placeholder="Select severities"
                                removeSelected
                                value={this.getFilterParams().severity}
                                className="text-base-600 font-400 min-w-64"
                            />
                        </div>
                    </div>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow border-t border-primary-300 bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderPolicyAlertsPanel()}
                        {this.renderModal()}
                    </div>
                </div>
            </section>
        );
    }
}

export default ViolationsPage;

import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import Table from 'Components/Table';
import MultiSelect from 'react-select';

import PolicyAlertsSidePanel from 'Containers/Violations/PolicyAlertsSidePanel';

import axios from 'axios';
import queryString from 'query-string';
import isEqual from 'lodash/isEqual';

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

const setSeverityClass = (item) => {
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
        default:
            return prevState;
    }
};

class ViolationsPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired
    }

    constructor(props) {
        super(props);

        this.pollTimeoutId = null;

        this.state = {
            category: {
                options: [{ label: 'Image Assurance', value: 'IMAGE_ASSURANCE' }, { label: 'Container Configuration', value: 'CONTAINER_CONFIGURATION' }, { label: 'Privileges & Capabilities', value: 'PRIVILEGES_CAPABILITIES' }]
            },
            alertsByPolicies: [],
            policy: {}
        };
    }

    componentDidMount() {
        this.pollAlertGroups();
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

    onCategoryFilterChange = (categories) => {
        this.props.history.push({
            pathname: this.props.location.pathname,
            search: queryString.stringify({
                ...this.getFilterParams(),
                category: categories.map(c => c.value)
            })
        });

        // history will be updated asynchronously and it should happen before alerts fetching
        // TODO-ivan: to be removed with switching to react-router-redux
        setTimeout(this.getAlertsGroups, 0);
    }

    getAlertsGroups = () => {
        const params = queryString.stringify({
            ...this.getFilterParams(),
            stale: false
        });
        return axios.get(`/v1/alerts/groups?${params}`).then((response) => {
            if (!response.data.alertsByPolicies ||
                isEqual(response.data.alertsByPolicies, this.state.alertsByPolicies)) return;
            this.setState({ alertsByPolicies: response.data.alertsByPolicies });
        }).catch((error) => {
            console.error(error);
        });
    }

    getFilterParams() {
        const { search } = this.props.location;
        const params = queryString.parse(search);
        return params;
    }

    pollAlertGroups = () => {
        this.getAlertsGroups().then(() => {
            this.pollTimeoutId = setTimeout(this.pollAlertGroups, 5000);
        });
    }

    openPanel = (policy) => {
        this.update('OPEN_POLICY_ALERTS_PANEL', { policy });
    }

    closePanel = () => {
        this.update('CLOSE_POLICY_ALERTS_PANEL');
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderTable = () => {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'description', label: 'Description' },
            {
                key: 'categories',
                label: 'Categories',
                keyValueFunc: obj => ((obj.length > 1) ? 'Multiple' : policyCategoriesLabels[obj[0]]),
                tooltip: categories => categories.map(category => policyCategoriesLabels[category]).join(' | ')
            },
            { key: 'severity', label: 'Severity', classFunc: setSeverityClass },
            { key: 'numAlerts', label: 'Alerts', align: 'right' }
        ];
        const rows = this.state.alertsByPolicies.map((obj) => {
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
    }

    renderPolicyAlertsPanel = () => {
        if (!this.state.isPanelOpen) return '';
        return <PolicyAlertsSidePanel policy={this.state.policy} onClose={this.closePanel} />;
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 flex-none">
                        <div className="flex flex-1 self-center justify-start">
                            <input
                                className="border rounded w-full p-3  border-base-300"
                                placeholder="Filter by registry, severity, deployment, or tag"
                            />
                        </div>
                        <div className="flex self-center justify-end ml-3">
                            <MultiSelect
                                multi
                                onChange={this.onCategoryFilterChange}
                                options={this.state.category.options}
                                placeholder="Select categories"
                                removeSelected
                                value={this.getFilterParams().category}
                                className="text-base-600 font-400 min-w-64"
                            />
                        </div>
                    </div>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow border-t border-primary-300 bg-base-100">
                            {this.renderTable()}
                        </div>
                        {this.renderPolicyAlertsPanel()}
                    </div>
                </div>
            </section>
        );
    }
}

export default ViolationsPage;

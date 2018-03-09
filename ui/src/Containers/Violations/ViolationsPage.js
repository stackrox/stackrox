import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import MultiSelect from 'react-select';
import queryString from 'query-string';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { sortNumber, sortSeverity } from 'sorters/sorters';
import { actions as alertActions } from 'reducers/alerts';
import { selectors } from 'reducers';

import Table from 'Components/Table';
import PolicyAlertsSidePanel from './PolicyAlertsSidePanel';
import ViolationsModal from './ViolationsModal';

const severityLabels = {
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low'
};

const getSeverityClassName = severityValue => {
    const severityClassMapping = {
        Low: 'text-low-500',
        Medium: 'text-medium-500',
        High: 'text-high-500',
        Critical: 'text-critical-500'
    };
    const res = severityClassMapping[severityValue];
    if (res) return res;
    throw new Error(`Unknown severity: ${severityValue}`);
};

const categoryOptions = [
    { label: 'Image Assurance', value: 'Image Assurance' },
    { label: 'Container Configuration', value: 'Container Configuration' },
    { label: 'Privileges & Capabilities', value: 'Privileges & Capabilities' }
];

const severityOptions = [
    { label: 'Critical Severity', value: 'CRITICAL_SEVERITY' },
    { label: 'High Severity', value: 'HIGH_SEVERITY' },
    { label: 'Medium Severity', value: 'MEDIUM_SEVERITY' },
    { label: 'Low Severity', value: 'LOW_SEVERITY' }
];

class ViolationsPage extends Component {
    static propTypes = {
        violatedPolicies: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
                numAlerts: PropTypes.string.isRequired
            })
        ).isRequired,
        alertsForSelectedPolicy: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        selectedPolicy: PropTypes.shape({
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired
        }),
        selectViolatedPolicy: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.match.isRequired
    };

    static defaultProps = {
        selectedPolicy: null
    };

    onFilterChange = type => options => {
        this.props.history.push({
            pathname: this.props.location.pathname,
            search: queryString.stringify({
                ...this.getFilterParams(),
                [type]: options.map(c => c.value)
            })
        });
    };

    onViolatedPolicyClick = policy => {
        this.props.selectViolatedPolicy(policy.id);
    };

    onCloseSidePanel = () => {
        this.props.selectViolatedPolicy(null);
    };

    onAlertClick = alert => {
        this.updateAlertUrl(alert.id);
    };

    onAlertModalClose = () => {
        this.updateAlertUrl();
    };

    getFilterParams() {
        const { search } = this.props.location;
        const params = queryString.parse(search);
        return params;
    }

    updateAlertUrl(alertId) {
        const urlSuffix = alertId ? `/${alertId}` : '';
        this.props.history.push({
            pathname: `/main/violations${urlSuffix}`,
            search: this.props.location.search
        });
    }

    renderTable() {
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'description', label: 'Description' },
            {
                key: 'categories',
                label: 'Categories',
                keyValueFunc: obj => (obj.length > 1 ? 'Multiple' : obj[0]),
                tooltip: categories => categories.join(' | ')
            },
            {
                key: 'severity',
                label: 'Severity',
                classFunc: getSeverityClassName,
                sortMethod: sortSeverity
            },
            {
                key: 'numAlerts',
                label: 'Alerts',
                align: 'right',
                sortMethod: sortNumber('numAlerts')
            }
        ];
        const rows = this.props.violatedPolicies.map(policy => ({
            ...policy,
            severity: severityLabels[policy.severity]
        }));
        return <Table columns={columns} rows={rows} onRowClick={this.onViolatedPolicyClick} />;
    }

    renderModal() {
        if (!this.props.match.params.alertId) return null;
        return (
            <ViolationsModal
                alertId={this.props.match.params.alertId}
                onClose={this.onAlertModalClose}
            />
        );
    }

    renderPolicyAlertsPanel() {
        if (!this.props.selectedPolicy) return null;
        return (
            <PolicyAlertsSidePanel
                header={this.props.selectedPolicy.name}
                alerts={this.props.alertsForSelectedPolicy}
                onClose={this.onCloseSidePanel}
                onRowClick={this.onAlertClick}
            />
        );
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 self-end justify-end">
                        <div className="flex ml-3">
                            <MultiSelect
                                multi
                                onChange={this.onFilterChange('category')}
                                options={categoryOptions}
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
                                options={severityOptions}
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

const getViolatedPolicies = createSelector(
    [selectors.getPoliciesById, selectors.getAlertNumsByPolicy],
    (policiesById, alertNumsByPolicy) =>
        alertNumsByPolicy.map(alertNum => ({
            ...policiesById[alertNum.policy],
            numAlerts: alertNum.numAlerts
        }))
);

const getAlertsForSelectedPolicy = createSelector(
    [selectors.getAlertsById, selectors.getSelectedViolatedPolicyId, selectors.getAlertsByPolicy],
    (alerts, policyId, alertsByPolicy) => {
        if (!policyId || !alertsByPolicy[policyId]) return [];
        return alertsByPolicy[policyId].map(alertId => alerts[alertId]);
    }
);

const getSelectedPolicy = createSelector(
    [selectors.getPoliciesById, selectors.getSelectedViolatedPolicyId],
    (policiesById, policyId) => policiesById[policyId]
);

const mapStateToProps = createStructuredSelector({
    violatedPolicies: getViolatedPolicies,
    alertsForSelectedPolicy: getAlertsForSelectedPolicy,
    selectedPolicy: getSelectedPolicy
});

const mapDispatchToProps = dispatch => ({
    selectViolatedPolicy: policyId => dispatch(alertActions.selectViolatedPolicy(policyId))
});

export default connect(mapStateToProps, mapDispatchToProps)(ViolationsPage);

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { sortNumber, sortSeverity } from 'sorters/sorters';
import { actions as alertActions } from 'reducers/alerts';
import { selectors } from 'reducers';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
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
        match: ReactRouterPropTypes.match.isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired
    };

    static defaultProps = {
        selectedPolicy: null
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
                label: 'Violations',
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
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 flex-col">
                    <PageHeader header="Violations" subHeader={subHeader}>
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

const isViewFiltered = createSelector(
    [selectors.getAlertsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    violatedPolicies: getViolatedPolicies,
    alertsForSelectedPolicy: getAlertsForSelectedPolicy,
    selectedPolicy: getSelectedPolicy,
    searchOptions: selectors.getAlertsSearchOptions,
    searchModifiers: selectors.getAlertsSearchModifiers,
    searchSuggestions: selectors.getAlertsSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = dispatch => ({
    selectViolatedPolicy: policyId => dispatch(alertActions.selectViolatedPolicy(policyId)),
    setSearchOptions: searchOptions => dispatch(alertActions.setAlertsSearchOptions(searchOptions)),
    setSearchModifiers: searchModifiers =>
        dispatch(alertActions.setAlertsSearchModifiers(searchModifiers)),
    setSearchSuggestions: searchSuggestions =>
        dispatch(alertActions.setAlertsSearchSuggestions(searchSuggestions))
});

export default connect(mapStateToProps, mapDispatchToProps)(ViolationsPage);

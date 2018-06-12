import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { sortTime, sortSeverity } from 'sorters/sorters';
import { actions as alertActions } from 'reducers/alerts';
import { selectors } from 'reducers';
import dateFns from 'date-fns';

import NoResultsMessage from 'Components/NoResultsMessage';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Table from 'Components/Table';
import ViolationsPanel from './ViolationsPanel';

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
                id: PropTypes.string.isRequired
            })
        ).isRequired,
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

    onViolationClick = alert => {
        this.updateAlertUrl(alert.id);
    };

    onPanelClose = () => {
        this.updateAlertUrl();
    };

    updateAlertUrl(id) {
        const urlSuffix = id ? `/${id}` : '';
        this.props.history.push({
            pathname: `/main/violations${urlSuffix}`,
            search: this.props.location.search
        });
    }

    renderTable() {
        const columns = [
            { key: 'deployment.name', label: 'Deployment' },
            { key: 'deployment.clusterName', label: 'Cluster' },
            { key: 'policy.name', label: 'Violation' },
            { key: 'policy.description', label: 'Description' },
            {
                key: 'policy.categories',
                label: 'Categories',
                keyValueFunc: obj => (obj.length > 1 ? 'Multiple' : obj[0]),
                tooltip: categories => categories.join(' | ')
            },
            {
                key: 'policy.severity',
                keyValueFunc: severity => severityLabels[severity],
                label: 'Severity',
                classFunc: getSeverityClassName,
                sortMethod: sortSeverity
            },
            {
                key: 'time',
                keyValueFunc: time =>
                    `${dateFns.format(time, 'MM/DD/YYYY')} ${dateFns.format(time, 'h:mm:ss A')}`,
                label: 'Time',
                sortMethod: sortTime
            }
        ];
        const rows = this.props.violatedPolicies;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return <Table columns={columns} rows={rows} onRowClick={this.onViolationClick} />;
    }

    renderSidePanel = () => {
        if (!this.props.match.params.alertId) return null;
        return (
            <ViolationsPanel
                key={this.props.match.params.alertId}
                alertId={this.props.match.params.alertId}
                onClose={this.onPanelClose}
            />
        );
    };

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
                        {this.renderSidePanel()}
                    </div>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getAlertsSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    violatedPolicies: selectors.getFilteredAlerts,
    searchOptions: selectors.getAlertsSearchOptions,
    searchModifiers: selectors.getAlertsSearchModifiers,
    searchSuggestions: selectors.getAlertsSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = (dispatch, props) => ({
    setSearchOptions: searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            props.history.push('/main/violations');
        }
        dispatch(alertActions.setAlertsSearchOptions(searchOptions));
    },
    setSearchModifiers: alertActions.setAlertsSearchModifiers,
    setSearchSuggestions: alertActions.setAlertsSearchSuggestions
});

export default connect(mapStateToProps, mapDispatchToProps)(ViolationsPage);

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { sortDate, sortSeverity } from 'sorters/sorters';
import { actions as alertActions } from 'reducers/alerts';
import { selectors } from 'reducers';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import ReactTooltip from 'react-tooltip';

import NoResultsMessage from 'Components/NoResultsMessage';
import ReactRowSelectTable from 'Components/ReactRowSelectTable';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
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
        violations: PropTypes.arrayOf(
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

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/violations');
        }
    };

    onPanelClose = () => {
        this.updateSelectedAlert();
    };

    updateSelectedAlert = alert => {
        const urlSuffix = alert && alert.id ? `/${alert.id}` : '';
        this.props.history.push({
            pathname: `/main/violations${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderTable() {
        const columns = [
            {
                Header: 'Deployment',
                accessor: 'deployment.name',
                widthClassName: 'w-1/8',
                wrap: true
            },
            {
                Header: 'Cluster',
                accessor: 'deployment.clusterName',
                widthClassName: 'w-1/8',
                wrap: true
            },
            {
                Header: 'Policy',
                accessor: 'policy.name',
                widthClassName: 'w-1/8',
                wrap: true
            },
            {
                Header: 'Description',
                accessor: 'policy.description',
                widthClassName: 'w-1/4',
                wrap: true
            },
            {
                Header: 'Categories',
                accessor: 'policy.categories',
                widthClassName: 'w-1/8',
                wrap: true,
                Cell: ci =>
                    ci.value.length > 1 ? (
                        <div data-tip data-for="button-violation-categories">
                            Multiple
                            <ReactTooltip
                                id="button-violation-categories"
                                type="dark"
                                effect="solid"
                            >
                                {ci.value.join(' | ')}
                            </ReactTooltip>
                        </div>
                    ) : (
                        ci.value[0]
                    )
            },
            {
                Header: 'Severity',
                accessor: 'policy.severity',
                widthClassName: 'w-1/8',
                Cell: ci => {
                    const severity = severityLabels[ci.value];
                    return <span className={getSeverityClassName(severity)}>{severity}</span>;
                },
                sortMethod: sortSeverity
            },
            {
                Header: 'Time',
                accessor: 'time',
                widthClassName: 'w-1/8',
                wrap: true,
                Cell: ci => dateFns.format(ci.value, dateTimeFormat),
                sortMethod: sortDate
            }
        ];
        const rows = this.props.violations;
        const id = this.props.match.params.alertId;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <ReactRowSelectTable
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedAlert}
                selectedRowId={id}
                noDataText="No results found. Please refine your search."
            />
        );
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
                            className="flex flex-1"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            onSearch={this.onSearch}
                        />
                    </PageHeader>
                    <div className="flex flex-1">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow border-primary-300 bg-base-100">
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
    violations: selectors.getFilteredAlerts,
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

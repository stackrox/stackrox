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
import Tooltip from 'rc-tooltip';

import { severityLabels, lifecycleStageLabels } from 'messages/common';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table, {
    pageSize,
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import ViolationsPanel from './ViolationsPanel';

const getSeverityClassName = severityValue => {
    const severityClassMapping = {
        Low: 'px-2 rounded-full bg-base-200 border-2 border-base-300 text-base-600',
        Medium: 'px-2 rounded-full bg-warning-200 border-2 border-warning-300 text-warning-800',
        High: 'px-2 rounded-full bg-caution-200 border-2 border-caution-300 text-caution-800',
        Critical: 'px-2 rounded-full bg-alert-200 border-2 border-alert-300 text-alert-800'
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

    constructor(props) {
        super(props);
        this.state = {
            page: 0
        };
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.props.history.push('/main/violations');
        }
    };

    onPanelClose = () => {
        this.updateSelectedAlert();
    };

    setTablePage = newPage => {
        this.setState({ page: newPage });
    };

    updateSelectedAlert = alert => {
        const urlSuffix = alert && alert.id ? `/${alert.id}` : '';
        this.props.history.push({
            pathname: `/main/violations${urlSuffix}`,
            search: this.props.location.search
        });
    };

    renderPanel = () => {
        const { length } = this.props.violations;
        const totalPages = length === pageSize ? 1 : Math.floor(length / pageSize) + 1;
        const paginationComponent = (
            <TablePagination
                page={this.state.page}
                totalPages={totalPages}
                setPage={this.setTablePage}
            />
        );
        const headerText = `${length} Violation${length === 1 ? '' : 's'} ${
            this.props.isViewFiltered ? 'Matched' : ''
        }`;
        return (
            <Panel header={headerText} headerComponents={paginationComponent}>
                <div className="w-full">{this.renderTable()}</div>
            </Panel>
        );
    };

    renderTable = () => {
        const columns = [
            {
                Header: 'Deployment',
                accessor: 'deployment.name',
                headerClassName: ` ${defaultHeaderClassName}`,
                className: ` ${wrapClassName} ${defaultColumnClassName}`
            },
            {
                Header: 'Cluster',
                accessor: 'deployment.clusterName',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`
            },
            {
                Header: 'Policy',
                accessor: 'policy.name',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ original }) => (
                    <div>
                        <Tooltip
                            placement="top"
                            mouseLeaveDelay={0}
                            overlay={<div>{original.policy.description}</div>}
                            overlayClassName="pointer-events-none text-white rounded max-w-xs p-2 w-full text-sm text-center"
                        >
                            <span className="inline-flex hover:text-primary-700 underline">
                                {original.policy.name}
                            </span>
                        </Tooltip>
                    </div>
                )
            },
            {
                Header: 'Severity',
                accessor: 'policy.severity',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => {
                    const severity = severityLabels[value];
                    return <span className={getSeverityClassName(severity)}>{severity}</span>;
                },
                sortMethod: sortSeverity
            },
            {
                Header: 'Categories',
                accessor: 'policy.categories',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) =>
                    value.length > 1 ? (
                        <Tooltip
                            placement="top"
                            mouseLeaveDelay={0}
                            overlay={<div>{value.join(' | ')}</div>}
                            overlayClassName="pointer-events-none text-white rounded max-w-xs p-2 w-full text-sm text-center"
                        >
                            <div>Multiple</div>
                        </Tooltip>
                    ) : (
                        value[0]
                    )
            },
            {
                Header: 'Lifecycle',
                accessor: 'policy.lifecycleStage',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => lifecycleStageLabels[value]
            },
            {
                Header: 'Time',
                accessor: 'time',
                headerClassName: `w-1/7 ${defaultHeaderClassName}`,
                className: `w-1/7 ${wrapClassName} ${defaultColumnClassName}`,
                Cell: ({ value }) => dateFns.format(value, dateTimeFormat),
                sortMethod: sortDate
            }
        ];
        const rows = this.props.violations;
        const id = this.props.match.params.alertId;
        if (!rows.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <Table
                rows={rows}
                columns={columns}
                onRowClick={this.updateSelectedAlert}
                selectedRowId={id}
                noDataText="No results found. Please refine your search."
                page={this.state.page}
            />
        );
    };

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
                        <div className="w-full rounded-sm shadow border-primary-300">
                            {this.renderPanel()}
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

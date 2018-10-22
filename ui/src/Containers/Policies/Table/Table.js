import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { pageSize } from 'Components/Table';
import { actions } from 'reducers/policies/table';

import Panel from 'Components/Panel';
import NoResultsMessage from 'Components/NoResultsMessage';
import TablePagination from 'Components/TablePagination';

import Buttons from 'Containers/Policies/Table/Buttons';
import TableContents from 'Containers/Policies/Table/TableContents';

// Table is the heading line with options to reasses and add a policy, as well as the underlying policy
// rows.
class Table extends Component {
    static propTypes = {
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        page: PropTypes.number.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,

        setPage: PropTypes.func.isRequired
    };

    pagination = (policies, page) => {
        const { length } = policies;
        const totalPages = Math.floor((length - 1) / pageSize) + 1;
        return <TablePagination page={page} totalPages={totalPages} setPage={this.props.setPage} />;
    };

    getTableHeaderText = () => {
        const selectionCount = this.props.selectedPolicyIds.length;
        const rowCount = this.props.policies.length;
        return selectionCount !== 0
            ? `${selectionCount} ${selectionCount === 1 ? 'Policy' : 'Policies'} Selected`
            : `${rowCount} ${rowCount === 1 ? 'Policy' : 'Policies'} ${
                  this.props.isViewFiltered ? 'Matched' : ''
              }`;
    };

    render() {
        if (!this.props.policies.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;

        return (
            <Panel
                header={this.getTableHeaderText()}
                buttons={<Buttons />}
                headerComponents={this.pagination(this.props.policies, this.props.page)}
            >
                <TableContents />
            </Panel>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getPoliciesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    policies: selectors.getFilteredPolicies,
    page: selectors.getTablePage,
    isViewFiltered
});

const mapDispatchToProps = {
    setPage: actions.setTablePage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Table);

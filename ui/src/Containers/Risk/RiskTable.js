import React from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

import NoResultsMessage from 'Components/NoResultsMessage';
import Table from 'Components/Table';

import { sortValue, sortDate } from 'sorters/sorters';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

const RiskTable = ({ history, location, rows, selectedDeployment, page }) => {
    function updateSelectedDeployment(deployment) {
        const urlSuffix = deployment && deployment.id ? `/${deployment.id}` : '';
        history.push({
            pathname: `/main/risk${urlSuffix}`,
            search: location.search
        });
    }

    const columns = [
        {
            Header: 'Name',
            accessor: 'name'
        },
        {
            id: 'updated',
            Header: 'Updated',
            accessor: 'updatedAt',
            // eslint-disable-next-line react/prop-types
            Cell: ({ value }) => <span>{dateFns.format(value, dateTimeFormat)}</span>,
            sortMethod: sortDate
        },
        {
            Header: 'Cluster',
            accessor: 'cluster'
        },
        {
            Header: 'Namespace',
            accessor: 'namespace'
        },
        {
            Header: 'Priority',
            accessor: 'priority',
            sortMethod: sortValue
        }
    ];

    const id = selectedDeployment && selectedDeployment.id;
    if (!rows.length)
        return <NoResultsMessage message="No results found. Please refine your search." />;
    return (
        <Table
            rows={rows}
            columns={columns}
            onRowClick={updateSelectedDeployment}
            selectedRowId={id}
            noDataText="No results found. Please refine your search."
            page={page}
        />
    );
};

RiskTable.propTypes = {
    rows: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired
    }),
    processGroup: PropTypes.shape({}),
    page: PropTypes.number.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

RiskTable.defaultProps = {
    selectedDeployment: null,
    processGroup: {}
};

export default withRouter(RiskTable);

import React from 'react';
import PropTypes from 'prop-types';
import VIOLATIONS from 'queries/violation';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import resolvePath from 'object-resolve-path';
import URLService from 'modules/URLService';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import uniq from 'lodash/uniq';
import { sortSeverity } from 'sorters/sorters';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import Table, { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import SeverityLabel from 'Components/SeverityLabel';
import TableWidget from './TableWidget';

const getDeploymentsGroupedByPolicies = data => {
    const { violations } = data;
    if (!violations || !violations.length) return [];
    const groups = violations.reduce((acc, curr) => {
        const { deployment, time, policy } = curr;
        const deployments = acc[policy.id] ? acc[policy.id].deployments : [];
        acc[policy.id] = {
            ...policy,
            deployments: [...deployments, { time, ...deployment }]
        };
        return acc;
    }, {});
    return Object.values(groups);
};

const Deployments = ({ original: policy, match, location, history }) => {
    const { deployments } = policy;
    const columns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id'
        },
        {
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name'
        },
        {
            Header: `Last Updated`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'time',
            Cell: ({ original }) => {
                const { time } = original;
                if (!time) return null;
                return format(time, dateTimeFormat);
            }
        }
    ];
    function onRowClick(row) {
        const id = resolvePath(row, 'id');
        const url = URLService.getURL(match, location)
            .push(entityTypes.DEPLOYMENT, id)
            .url();
        history.push(url);
    }
    return (
        <Table
            rows={deployments}
            columns={columns}
            onRowClick={onRowClick}
            idAttribute="id"
            id="capture-list"
            noDataText="No results found. Please refine your search."
        />
    );
};

Deployments.propTypes = {
    original: PropTypes.shape({
        deployments: PropTypes.arrayOf().isRequired
    }).isRequired,
    match: PropTypes.string.isRequired,
    location: PropTypes.string.isRequired,
    history: PropTypes.string.isRequired
};

const DeploymentsWithFailedPolicies = ({ query }) => {
    return (
        <Query query={VIOLATIONS} variables={{ query }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data) return null;
                const groups = getDeploymentsGroupedByPolicies(data);
                const numDeployments = uniq(data.violations.map(violation => violation.deployment))
                    .length;
                const header = `${numDeployments} deployments with failed policies`;
                const groupColumns = [
                    {
                        expander: true,
                        headerClassName: `w-1/8 ${defaultHeaderClassName} pointer-events-none`,
                        className: 'w-1/8 pointer-events-none flex items-center justify-end',
                        // eslint-disable-next-line react/prop-types
                        Expander: ({ isExpanded, ...rest }) => {
                            if (rest.original.deployments.length === 0) return '';
                            const className = 'rt-expander w-1 pt-2 pointer-events-auto';
                            return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
                        }
                    },
                    {
                        Header: 'Id',
                        headerClassName: 'hidden',
                        className: 'hidden',
                        accessor: 'id'
                    },
                    {
                        Header: `Policy`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        accessor: 'name'
                    },
                    {
                        Header: `Severity`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        // eslint-disable-next-line
                        Cell: ({ original }) => {
                            const { severity } = original;
                            return <SeverityLabel severity={severity} />;
                        },
                        accessor: 'severity',
                        sortMethod: sortSeverity
                    },
                    {
                        Header: `Categories`,
                        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                        className: `w-1/8 ${defaultColumnClassName}`,
                        Cell: ({ original }) => {
                            const { categories } = original;
                            return categories.join(', ');
                        },
                        accessor: 'lifecycleStages'
                    }
                ];
                return (
                    <TableWidget
                        header={header}
                        rows={groups}
                        noDataText="No Nodes"
                        className="bg-base-100 w-full"
                        columns={groupColumns}
                        SubComponent={withRouter(Deployments)}
                        idAttribute="id"
                    />
                );
            }}
        </Query>
    );
};

DeploymentsWithFailedPolicies.propTypes = {
    query: PropTypes.string
};

DeploymentsWithFailedPolicies.defaultProps = {
    query: ''
};

export default DeploymentsWithFailedPolicies;

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
import CollapsibleRow from 'Components/CollapsibleRow';
import NoResultsMessage from 'Components/NoResultsMessage';

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
            noDataText="No results found. Please refine your search."
        />
    );
};

Deployments.propTypes = {
    original: PropTypes.shape({}).isRequired,
    match: PropTypes.string.isRequired,
    location: PropTypes.string.isRequired,
    history: PropTypes.string.isRequired
};

const DeploymentsWithRouter = withRouter(Deployments);

const DeploymentsWithFailedPolicies = ({ query, message }) => {
    return (
        <Query query={VIOLATIONS} variables={{ query }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data) return null;
                const groups = getDeploymentsGroupedByPolicies(data);
                const numDeployments = uniq(data.violations.map(violation => violation.deployment))
                    .length;
                if (numDeployments === 0)
                    return (
                        <NoResultsMessage message={message} className="p-6 shadow" icon="info" />
                    );
                const header = `${numDeployments} deployments failed across ${
                    groups.length
                } policies`;
                const columns = [
                    {
                        Header: `Policy`,
                        headerClassName: `${defaultHeaderClassName} hidden`,
                        className: `${defaultColumnClassName} hidden`,
                        accessor: 'name',
                        Cell: ({ original }) => {
                            const { severity, categories, name } = original;

                            const groupHeader = (
                                <div className="flex flex-1">
                                    <div className="flex flex-1">{name}</div>
                                    <div>
                                        <span>
                                            Severity: <SeverityLabel severity={severity} />
                                        </span>
                                        <span className="pl-2 pr-2">|</span>
                                        <span>Categories: {categories.join(',')}</span>
                                    </div>
                                </div>
                            );
                            const group = (
                                <CollapsibleRow
                                    key={name}
                                    header={groupHeader}
                                    isCollapsibleOpen={false}
                                    className="z-20"
                                    hasTitleBorder={false}
                                >
                                    <DeploymentsWithRouter original={original} />
                                </CollapsibleRow>
                            );
                            return group;
                        }
                    }
                ];
                return (
                    <TableWidget
                        header={header}
                        rows={groups}
                        noDataText="No Nodes"
                        className="bg-base-100 w-full"
                        columns={columns}
                        idAttribute="id"
                        hasNestedTable
                    />
                );
            }}
        </Query>
    );
};

DeploymentsWithFailedPolicies.propTypes = {
    query: PropTypes.string,
    message: PropTypes.string
};

DeploymentsWithFailedPolicies.defaultProps = {
    query: '',
    message: ''
};

export default DeploymentsWithFailedPolicies;

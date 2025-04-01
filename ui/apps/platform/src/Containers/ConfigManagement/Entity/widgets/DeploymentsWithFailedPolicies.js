import React from 'react';
import PropTypes from 'prop-types';

import VIOLATIONS from 'queries/violation';
import resolvePath from 'object-resolve-path';
import URLService from 'utils/URLService';
import entityTypes from 'constants/entityTypes';
import uniq from 'lodash/uniq';
import CollapsibleRow from 'Components/CollapsibleRow';
import NoResultsMessage from 'Components/NoResultsMessage';

import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import { entityViolationsColumns } from 'constants/listColumns';
import useWorkflowMatch from 'hooks/useWorkflowMatch';

import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import Table, { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { useLocation, useNavigate } from 'react-router-dom';
import TableWidget from './TableWidget';

const getDeploymentsGroupedByPolicies = (data) => {
    const { violations } = data;
    if (!violations || !violations.length) {
        return [];
    }
    const groups = violations.reduce((acc, curr) => {
        const { deployment, time, policy } = curr;
        const deployments = acc[policy.id] ? acc[policy.id].deployments : [];
        acc[policy.id] = {
            ...policy,
            deployments: [...deployments, { time, ...deployment }],
        };
        return acc;
    }, {});
    return Object.values(groups);
};

const Deployments = ({ original: policy, entityContext }) => {
    const { deployments } = policy;
    const columns = entityViolationsColumns[entityTypes.DEPLOYMENT](entityContext);
    const navigate = useNavigate();
    const location = useLocation();
    const match = useWorkflowMatch();

    function onRowClick(row) {
        const id = resolvePath(row, 'id');
        const url = URLService.getURL(match, location).push(entityTypes.DEPLOYMENT, id).url();
        navigate(url);
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
    original: PropTypes.shape({ deployments: PropTypes.arrayOf(PropTypes.shape({})) }).isRequired,
    entityContext: PropTypes.shape({}),
};

Deployments.defaultProps = {
    entityContext: {},
};

const DeploymentsWithFailedPolicies = ({ query, message, entityContext }) => (
    <Query query={VIOLATIONS} variables={{ query }}>
        {({ loading, data }) => {
            if (loading) {
                return <Loader />;
            }
            if (!data) {
                return null;
            }
            const groups = getDeploymentsGroupedByPolicies(data);
            const numDeployments = uniq(
                data.violations.map((violation) => violation.deployment)
            ).length;
            if (numDeployments === 0) {
                return <NoResultsMessage message={message} className="p-3 shadow" icon="info" />;
            }
            const header = `${numDeployments} deployments failed across ${groups.length} policies`;
            const columns = [
                {
                    Header: `Policy`,
                    headerClassName: defaultHeaderClassName,
                    className: defaultColumnClassName,
                    accessor: 'name',
                    Cell: ({ original, pdf }) => {
                        const { severity, categories, name } = original;

                        const groupHeader = (
                            <div className="flex flex-1">
                                <div className="flex flex-1">{name}</div>
                                <div>
                                    <span>
                                        Severity:{' '}
                                        <PolicySeverityIconText
                                            severity={severity}
                                            isTextOnly={pdf}
                                        />
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
                                <Deployments original={original} entityContext={entityContext} />
                            </CollapsibleRow>
                        );
                        return group;
                    },
                },
            ];
            return (
                <TableWidget
                    header={header}
                    rows={groups}
                    noDataText="No deployments failing across policies"
                    className="w-full"
                    columns={columns}
                    idAttribute="id"
                    id="deployments-with-failed-policies"
                    hasNestedTable
                />
            );
        }}
    </Query>
);

DeploymentsWithFailedPolicies.propTypes = {
    query: PropTypes.string,
    message: PropTypes.string,
    entityContext: PropTypes.shape({}),
};

DeploymentsWithFailedPolicies.defaultProps = {
    query: '',
    message: '',
    entityContext: {},
};

export default DeploymentsWithFailedPolicies;

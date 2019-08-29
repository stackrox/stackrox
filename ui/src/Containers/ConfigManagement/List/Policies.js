import React from 'react';
import entityTypes from 'constants/entityTypes';
import { POLICIES as QUERY } from 'queries/policy';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import { sortSeverity } from 'sorters/sorters';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import LifecycleStageLabel from 'Components/LifecycleStageLabel';
import SeverityLabel from 'Components/SeverityLabel';
import LabelChip from 'Components/LabelChip';
import List from './List';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

const tableColumns = [
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
        Header: `Enabled`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { disabled } = original;
            return disabled ? 'No' : 'Yes';
        },
        accessor: 'disabled'
    },
    {
        Header: `Enforced`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { enforcementActions } = original;
            return enforcementActions.length === 0 ||
                enforcementActions.includes('UNSET_ENFORCEMENT')
                ? 'No'
                : 'Yes';
        },
        accessor: 'enforcementActions'
    },
    {
        Header: `Policy Status`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        // eslint-disable-next-line
        Cell: ({ original }) => {
            const { policyStatus } = original;
            return policyStatus === 'pass' ? 'Pass' : <LabelChip text="Fail" type="alert" />;
        },
        accessor: 'policyStatus'
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
        accessor: 'categories'
    },
    {
        Header: `Lifecycle Stage`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { lifecycleStages } = original;
            return lifecycleStages.map(lifecycleStage => (
                <LifecycleStageLabel
                    key={lifecycleStage}
                    className="mr-2"
                    lifecycleStage={lifecycleStage}
                />
            ));
        },
        accessor: 'lifecycleStages'
    }
];

const createTableRows = data => data.policies;

const Policies = ({ className, onRowClick, query, selectedRowId, data }) => {
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } = query || {};
    const queryText = queryService.objectToWhereClause({
        'Lifecycle Stage': 'DEPLOY',
        ...restQuery
    });
    const variables = queryText ? { query: queryText } : null;

    function createTableRowsFilteredByPolicyStatus(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.POLICY}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            selectedRowId={selectedRowId}
            onRowClick={onRowClick}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'severity',
                    desc: false
                }
            ]}
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
        />
    );
};

Policies.propTypes = entityListPropTypes;
Policies.defaultProps = entityListDefaultprops;

export default Policies;

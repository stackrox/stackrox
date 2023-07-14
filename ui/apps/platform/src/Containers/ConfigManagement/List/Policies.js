import React from 'react';

import {
    defaultHeaderClassName,
    defaultColumnClassName,
    nonSortableHeaderClassName,
} from 'Components/Table';
import PolicyDisabledIconText from 'Components/PatternFly/IconText/PolicyDisabledIconText';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import entityTypes from 'constants/entityTypes';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { policySortFields } from 'constants/sortFields';
import { formatLifecycleStages } from 'Containers/Policies/policies.utils';
import { POLICIES_QUERY } from 'queries/policy';
import { sortSeverity } from 'sorters/sorters';
import queryService from 'utils/queryService';
import ListFrontendPaginated from './ListFrontendPaginated';

import filterByPolicyStatus from './utilities/filterByPolicyStatus';

export const defaultPolicyrSort = [
    {
        id: policySortFields.POLICY,
        desc: false,
    },
];

const tableColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id',
    },
    {
        Header: `Policy`,
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        accessor: 'name',
        id: policySortFields.POLICY,
        sortField: policySortFields.POLICY,
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
        accessor: 'enforcementActions',
        id: policySortFields.ENFORCEMENT,
        sortField: policySortFields.ENFORCEMENT,
    },
    {
        Header: `Policy Status`,
        headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original, pdf }) => {
            const { disabled, policyStatus } = original;
            return disabled ? (
                <PolicyDisabledIconText isDisabled={disabled} isTextOnly={pdf} />
            ) : (
                <PolicyStatusIconText isPass={policyStatus === 'pass'} isTextOnly={pdf} />
            );
        },
        accessor: 'policyStatus',
        sortable: false, // not performant as of 2020-06-11
    },
    {
        Header: `Severity`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { severity, pdf } = original;
            return <PolicySeverityIconText severity={severity} isTextOnly={pdf} />;
        },
        accessor: 'severity',
        sortMethod: sortSeverity,
        id: policySortFields.SEVERITY,
        sortField: policySortFields.SEVERITY,
    },
    {
        Header: `Categories`,
        headerClassName: `w-1/4 ${defaultHeaderClassName}`,
        className: `w-1/4 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { categories } = original;
            return categories.join(', ');
        },
        accessor: 'categories',
        id: policySortFields.CATEGORY,
        sortField: policySortFields.CATEGORY,
    },
    {
        Header: `Lifecycle Stage`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { lifecycleStages } = original;
            return formatLifecycleStages(lifecycleStages);
        },
        accessor: 'lifecycleStages',
        id: policySortFields.LIFECYCLE_STAGE,
        sortField: policySortFields.LIFECYCLE_STAGE,
    },
];

const createTableRows = (data) => data.policies;

const Policies = ({ className, onRowClick, query, selectedRowId, data }) => {
    const autoFocusSearchInput = !selectedRowId;
    const { [SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]: policyStatus, ...restQuery } = query || {};
    const queryText = queryService.objectToWhereClause({
        'Lifecycle Stage': 'DEPLOY',
        ...restQuery,
    });
    const variables = queryText ? { query: queryText } : null;

    function createTableRowsFilteredByPolicyStatus(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByPolicyStatus(tableRows, policyStatus);
        return filteredTableRows;
    }

    return (
        <ListFrontendPaginated
            className={className}
            query={POLICIES_QUERY}
            variables={variables}
            entityType={entityTypes.POLICY}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByPolicyStatus}
            selectedRowId={selectedRowId}
            onRowClick={onRowClick}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'policyStatus',
                    desc: false,
                },
                {
                    id: 'severity',
                    desc: false,
                },
            ]}
            defaultSearchOptions={[SEARCH_OPTIONS.POLICY_STATUS.CATEGORY]}
            data={filterByPolicyStatus(data, policyStatus)}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};

Policies.propTypes = entityListPropTypes;
Policies.defaultProps = entityListDefaultprops;

export default Policies;

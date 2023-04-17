import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { Power, Bell, BellOff, Trash2 } from 'react-feather';
import { connect } from 'react-redux';

import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import DateTimeField from 'Components/DateTimeField';
import Dialog from 'Components/Dialog';
import PolicyDisabledIconText from 'Components/PatternFly/IconText/PolicyDisabledIconText';
import PolicySeverityIconText from 'Components/PatternFly/IconText/PolicySeverityIconText';
import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import IconWithState from 'Components/IconWithState';
import PanelButton from 'Components/PanelButton';
import RowActionButton from 'Components/RowActionButton';
import TableCountLink from 'Components/workflow/TableCountLink';
import { formatLifecycleStages } from 'Containers/Policies/policies.utils';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import queryService from 'utils/queryService';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import { actions as notificationActions } from 'reducers/notifications';
import { deletePolicies } from 'services/PoliciesService';
import removeEntityContextColumns from 'utils/tableUtils';
import { policySortFields } from 'constants/sortFields';

import { UNSCOPED_POLICY_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';

export const defaultPolicySort = [
    // @TODO: remove this fake default sort on Policy name, when latest violation is available
    {
        id: policySortFields.POLICY,
        desc: false,
    },
    // {
    //     id: policySortFields.LATEST_VIOLATION,
    //     desc: true
    // }
];

export function getPolicyTableColumns(workflowState) {
    // to determine whether to show the counts as links in the table when not in pure POLICY state
    const inFindingsSection = workflowState.getCurrentEntity().entityType !== entityTypes.POLICY;
    const tableColumns = [
        {
            Header: 'id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: 'statuses',
            headerClassName: 'w-16 invisible',
            className: `w-16 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { disabled, notifiers } = original;
                const policyEnabled = !disabled;
                const hasNotifiers = !!(notifiers && notifiers.length);

                return (
                    <div className="flex">
                        <IconWithState Icon={Power} enabled={policyEnabled} />
                        <IconWithState
                            Icon={hasNotifiers ? Bell : BellOff}
                            enabled={hasNotifiers}
                        />
                    </div>
                );
            },
        },
        {
            Header: `Policy`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            id: policySortFields.POLICY,
            accessor: 'name',
            sortField: policySortFields.POLICY,
        },
        {
            Header: `Description`,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            id: policySortFields.DESCRIPTION,
            accessor: 'description',
            sortable: false,
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-24 ${nonSortableHeaderClassName}`,
            className: `w-24 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { disabled, policyStatus } = original;
                return disabled ? (
                    <PolicyDisabledIconText isDisabled={disabled} isTextOnly={pdf} />
                ) : (
                    <PolicyStatusIconText isPass={policyStatus === 'pass'} isTextOnly={pdf} />
                );
            },
            id: policySortFields.POLICY_STATUS,
            accessor: 'policyStatus',
            sortField: policySortFields.POLICY_STATUS,
            sortable: false,
        },
        {
            Header: `Last Updated`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { lastUpdated } = original;
                return <DateTimeField date={lastUpdated} asString={pdf} />;
            },
            id: policySortFields.LAST_UPDATED,
            accessor: 'lastUpdated',
            sortField: policySortFields.LAST_UPDATED,
        },
        {
            Header: `Latest Violation`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const { latestViolation } = original;
                return <DateTimeField date={latestViolation} asString={pdf} />;
            },
            id: policySortFields.LATEST_VIOLATION,
            accessor: 'latestViolation',
            sortField: policySortFields.LATEST_VIOLATION,
            sortable: false,
        },
        {
            Header: `Severity`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <PolicySeverityIconText severity={original.severity} isTextOnly={pdf} />
            ),
            id: policySortFields.SEVERITY,
            accessor: 'severity',
            sortField: policySortFields.SEVERITY,
        },
        {
            Header: `Deployments`,
            entityType: entityTypes.DEPLOYMENT,
            headerClassName: `w-1/8 ${nonSortableHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    entityTypeText="failing deployments"
                    count={original.deploymentCount}
                    textOnly={inFindingsSection || pdf}
                    selectedRowId={original.id}
                    // 'Policy Violated' is a fake search field to filter deployments that have violation. This is
                    // handled/supported only by deployments sub-resolver of policy resolver. Note that
                    // 'Policy Violated=false' is not yet supported. Refer to 'pkg/search/options.go' for details.
                    search={{ 'Policy Violated': true }}
                />
            ),
            id: policySortFields.DEPLOYMENTS,
            accessor: 'deploymentCount',
            sortField: policySortFields.DEPLOYMENTS,
            sortable: false, // not performant as of 2020-01-28
        },
        {
            Header: `Lifecycle`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { lifecycleStages } = original;
                if (!lifecycleStages || !lifecycleStages.length) {
                    return 'No lifecycle stages';
                }

                return <span>{formatLifecycleStages(lifecycleStages)}</span>;
            },
            id: policySortFields.LIFECYCLE_STAGE,
            accessor: 'lifecycleStages',
            sortField: policySortFields.LIFECYCLE_STAGE,
        },
        {
            Header: `Enforcement`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { enforcementActions } = original;
                return enforcementActions && enforcementActions.length ? 'Yes' : 'No';
            },
            id: policySortFields.ENFORCEMENT,
            accessor: 'enforcementActions',
            sortField: policySortFields.ENFORCEMENT,
        },
    ];

    return removeEntityContextColumns(tableColumns, workflowState);
}

const VulnMgmtPolicies = ({
    selectedRowId,
    search,
    sort,
    page,
    data,
    totalResults,
    addToast,
    removeToast,
    refreshTrigger,
    setRefreshTrigger,
}) => {
    const [selectedPolicyIds, setSelectedPolicyIds] = useState([]);
    const [bulkActionPolicyIds, setBulkActionPolicyIds] = useState([]);

    const POLICIES_QUERY = gql`
        query getPolicies($policyQuery: String, $scopeQuery: String, $pagination: Pagination) {
            results: policies(query: $policyQuery, pagination: $pagination) {
                ...unscopedPolicyFields
            }
            count: policyCount(query: $policyQuery)
        }
        ${UNSCOPED_POLICY_LIST_FRAGMENT}
    `;

    const tableSort = sort || defaultPolicySort;
    const queryOptions = {
        variables: {
            policyQuery: queryService.objectToWhereClause({
                ...search,
                Category: 'Vulnerability Management',
                cachebuster: refreshTrigger,
            }),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    const deletePolicy = (policyId) => (e) => {
        e.stopPropagation();

        const policiesToDelete = policyId ? [policyId] : selectedPolicyIds;

        if (policiesToDelete.length) {
            setBulkActionPolicyIds(policiesToDelete);
        } else {
            throw new Error('Logic error: tried to delete with no Policy IDs selected.');
        }
    };

    function hideDialog() {
        setBulkActionPolicyIds([]);
    }

    function makeDeleteRequest() {
        deletePolicies(bulkActionPolicyIds)
            .then(() => {
                setSelectedPolicyIds([]);

                // changing this param value on the query vars, to force the query to refetch
                setRefreshTrigger(Math.random());

                addToast(
                    `Successfully deleted ${bulkActionPolicyIds.length} ${pluralize(
                        'policy',
                        bulkActionPolicyIds.length
                    )}`
                );
                setTimeout(removeToast, 2000);

                hideDialog();
            })
            .catch((evt) => {
                addToast(`Could not delete all of the selected policies: ${evt.message}`);
                setTimeout(removeToast, 3000);
            });
    }

    const renderRowActionButtons = ({ id, isDefault }) => (
        <div className="flex border-2 border-r-2 border-base-400 bg-base-100">
            <RowActionButton
                text="Delete this Policy"
                onClick={deletePolicy(id)}
                icon={
                    <Trash2 className="hover:bg-alert-200 text-alert-600 hover:text-alert-700 my-1 h-4 w-4" />
                }
                disabled={isDefault}
            />
        </div>
    );

    const tableHeaderComponents = (
        <>
            <PanelButton
                icon={<Trash2 className="h-4 w-4" />}
                className="btn-icon btn-alert"
                onClick={deletePolicy()}
                disabled={selectedPolicyIds.length === 0}
                tooltip="Delete Selected Policies"
            >
                Delete
            </PanelButton>
        </>
    );

    return (
        <>
            <WorkflowListPage
                data={data}
                totalResults={totalResults}
                query={POLICIES_QUERY}
                queryOptions={queryOptions}
                idAttribute="id"
                entityListType={entityTypes.POLICY}
                getTableColumns={getPolicyTableColumns}
                selectedRowId={selectedRowId}
                search={search}
                sort={tableSort}
                page={page}
                checkbox
                tableHeaderComponents={tableHeaderComponents}
                selection={selectedPolicyIds}
                setSelection={setSelectedPolicyIds}
                renderRowActionButtons={renderRowActionButtons}
            />
            <Dialog
                className="w-1/3"
                isOpen={bulkActionPolicyIds.length > 0}
                text={`Are you sure you want to delete ${bulkActionPolicyIds.length} ${pluralize(
                    'policy',
                    bulkActionPolicyIds.length
                )}?`}
                onConfirm={makeDeleteRequest}
                confirmText="Delete"
                onCancel={hideDialog}
                isDestructive
            />
        </>
    );
};

VulnMgmtPolicies.propTypes = {
    ...workflowListPropTypes,
    refreshTrigger: PropTypes.number,
    setRefreshTrigger: PropTypes.func,
};
VulnMgmtPolicies.defaultProps = {
    ...workflowListDefaultProps,
    refreshTrigger: 0,
    setRefreshTrigger: null,
};

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(VulnMgmtPolicies);

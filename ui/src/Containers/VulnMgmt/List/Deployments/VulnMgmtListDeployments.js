import React, { useContext } from 'react';
import { PropTypes } from 'prop-types';
import pluralize from 'pluralize';
import gql from 'graphql-tag';

import queryService from 'modules/queryService';
import DateTimeField from 'Components/DateTimeField';
import FixableCVECount from 'Components/FixableCVECount';
import LabelChip from 'Components/LabelChip';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';
import TableCellLink from 'Components/TableCellLink';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import { getLatestDatedItemByKey } from 'utils/dateUtils';
import { getSeverityCounts } from 'utils/vulnerabilityUtils';
import { severities } from 'constants/severities';

export const defaultDeploymentSort = [
    {
        id: 'priority',
        desc: false
    }
];

const VulnMgmtDeployments = ({ selectedRowId, search, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    const query = gql`
        query getDeployments($query: String) {
            results: deployments(query: $query) {
                ...deploymentListFields
            }
        }
        ${DEPLOYMENT_LIST_FRAGMENT}
    `;

    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause(search)
        }
    };

    function getTableColumns() {
        const tableColumns = [
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
                Header: `CVEs`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnerabilities, id } = original;
                    if (!vulnerabilities || vulnerabilities.length === 0) return 'No CVEs';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(id).pushList(entityTypes.CVE);
                    const url = generateURL(workflowStateMgr.workflowState);

                    const fixables = vulnerabilities.filter(vuln => vuln.isFixable);
                    const counts = getSeverityCounts(vulnerabilities);
                    const tooltipBody = (
                        <div>
                            <div>
                                {counts[severities.CRITICAL_SEVERITY].total} Critical CVEs (
                                {counts[severities.CRITICAL_SEVERITY].fixable} Fixable)
                            </div>
                            <div>
                                {counts[severities.HIGH_SEVERITY].total} High CVEs (
                                {counts[severities.HIGH_SEVERITY].fixable} Fixable)
                            </div>
                            <div>
                                {counts[severities.MEDIUM_SEVERITY].total} Medium CVEs (
                                {counts[severities.MEDIUM_SEVERITY].fixable} Fixable)
                            </div>
                            <div>
                                {counts[severities.LOW_SEVERITY].total} Low CVEs (
                                {counts[severities.LOW_SEVERITY].fixable} Fixable)
                            </div>
                        </div>
                    );

                    return (
                        <div className="flex items-center">
                            <FixableCVECount
                                cves={vulnerabilities.length}
                                fixable={fixables.length}
                                orientation="vertical"
                                url={url}
                                pdf={pdf}
                            />
                            <SeverityStackedPill
                                critical={counts[severities.CRITICAL_SEVERITY].total}
                                high={counts[severities.HIGH_SEVERITY].total}
                                medium={counts[severities.MEDIUM_SEVERITY].total}
                                low={counts[severities.LOW_SEVERITY].total}
                                tooltip={{ title: 'Criticality Distribution', body: tooltipBody }}
                            />
                        </div>
                    );
                }
            },
            {
                Header: `Latest Violation`,
                headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                className: `w-1/8 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { deployAlerts } = original;
                    if (!deployAlerts || !deployAlerts.length) return '-';

                    const latestAlert = getLatestDatedItemByKey('time', deployAlerts);

                    return <DateTimeField date={latestAlert.time} />;
                }
            },
            entityContext[entityTypes.POLICY]
                ? null
                : {
                      Header: `Policies`,
                      headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                      className: `w-1/10 ${defaultColumnClassName}`,
                      accessor: 'failingPolicyCount',
                      Cell: ({ original, pdf }) => {
                          const { failingPolicyCount, id } = original;
                          if (failingPolicyCount === 0) return 'No failing policies';

                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr.pushListItem(id).pushList(entityTypes.POLICY);
                          const url = generateURL(workflowStateMgr.workflowState);
                          return (
                              <TableCellLink
                                  pdf={pdf}
                                  url={url}
                                  text={`${failingPolicyCount} ${pluralize(
                                      'policies',
                                      failingPolicyCount
                                  )}`}
                              />
                          );
                      }
                  },
            {
                Header: `Policy Status`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { policyStatus } = original;
                    return policyStatus === 'pass' ? (
                        <LabelChip text="Pass" type="success" />
                    ) : (
                        <LabelChip text="Fail" type="alert" />
                    );
                },
                id: 'policyStatus',
                accessor: 'policyStatus'
            },
            entityContext[entityTypes.CLUSTER]
                ? null
                : {
                      Header: `Cluster`,
                      headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                      className: `w-1/8 ${defaultColumnClassName}`,
                      accessor: 'clusterName',
                      Cell: ({ original, pdf }) => {
                          const { clusterName, clusterId, id } = original;
                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr
                              .pushListItem(id)
                              .pushRelatedEntity(entityTypes.CLUSTER, clusterId);
                          const url = generateURL(workflowStateMgr.workflowState);
                          return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
                      }
                  },
            entityContext[entityTypes.NAMESPACE]
                ? null
                : {
                      Header: `Namespace`,
                      headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                      className: `w-1/8 ${defaultColumnClassName}`,
                      accessor: 'namespace',
                      Cell: ({ original, pdf }) => {
                          const { namespace, namespaceId, id } = original;
                          const workflowStateMgr = new WorkflowStateMgr(workflowState);
                          workflowStateMgr
                              .pushListItem(id)
                              .pushRelatedEntity(entityTypes.NAMESPACE, namespaceId);
                          const url = generateURL(workflowStateMgr.workflowState);
                          return <TableCellLink pdf={pdf} url={url} text={namespace} />;
                      }
                  },
            {
                Header: `Images`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { imageCount, id } = original;
                    if (imageCount === 0) return 'No images';
                    const workflowStateMgr = new WorkflowStateMgr(workflowState);
                    workflowStateMgr.pushListItem(id).pushList(entityTypes.IMAGE);
                    const url = generateURL(workflowStateMgr.workflowState);
                    return (
                        <TableCellLink
                            pdf={pdf}
                            url={url}
                            text={`${imageCount} ${pluralize('image', imageCount)}`}
                        />
                    );
                },
                accessor: 'imageCount'
            },
            {
                Header: `Risk`,
                headerClassName: `w-1/10 ${defaultHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName}`,
                accessor: 'priority'
            }
        ];
        return tableColumns.filter(col => col);
    }

    return (
        <WorkflowListPage
            query={query}
            queryOptions={queryOptions}
            entityListType={entityTypes.DEPLOYMENT}
            getTableColumns={getTableColumns}
            defaultSorted={defaultDeploymentSort}
            selectedRowId={selectedRowId}
            search={search}
        />
    );
};

VulnMgmtDeployments.propTypes = {
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    entityContext: PropTypes.shape({})
};

VulnMgmtDeployments.defaultProps = {
    search: null,
    entityContext: {},
    selectedRowId: null
};

export default VulnMgmtDeployments;

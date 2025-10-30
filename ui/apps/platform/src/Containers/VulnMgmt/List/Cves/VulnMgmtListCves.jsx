import React, { useContext, useState } from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';
import { Plus } from 'react-feather';
import { connect } from 'react-redux';

import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import DateTimeField from 'Components/DateTimeField';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TableCellLink from 'Components/TableCellLink';
import TopCvssLabel from 'Components/TopCvssLabel';
import PanelButton from 'Components/PanelButton';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { actions as notificationActions } from 'reducers/notifications';
import queryService from 'utils/queryService';
import removeEntityContextColumns from 'utils/tableUtils';
import { cveSortFields } from 'constants/sortFields';
import {
    IMAGE_CVE_LIST_FRAGMENT,
    NODE_CVE_LIST_FRAGMENT,
    CLUSTER_CVE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';

import CveType from 'Components/CveType';

import CveBulkActionDialogue from './CveBulkActionDialogue';

import { getVulnMgmtPathForEntitiesAndId } from '../../VulnMgmt.utils/entities';
import WorkflowListPage from '../WorkflowListPage';
import { getFilteredCVEColumns } from './ListCVEs.utils';
import TableCountLinks from '../../TableCountLinks';

export const defaultCveSort = [
    {
        id: cveSortFields.CVSS_SCORE,
        desc: true,
    },
];

export function getCveTableColumns(workflowState, isFeatureFlagEnabled) {
    // to determine whether to show the counts as links in the table when not in pure CVE state
    const currentEntityType = workflowState.getCurrentEntity().entityType;
    const isCveType = [
        entityTypes.CVE, // TODO: remove this type after it's removed from workflow
        entityTypes.IMAGE_CVE,
        entityTypes.NODE_CVE,
        entityTypes.CLUSTER_CVE,
    ].includes(currentEntityType);
    const inFindingsSection = !isCveType;

    const tableColumns = [
        {
            expander: true,
            show: false,
        },
        {
            Header: 'id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `CVE`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                const url = getVulnMgmtPathForEntitiesAndId(currentEntityType, original.id);
                return (
                    <TableCellLink pdf={pdf} url={url}>
                        {original.cve}
                    </TableCellLink>
                );
            },
            id: cveSortFields.CVE,
            accessor: 'cve',
            sortField: cveSortFields.CVE,
        },
        {
            Header: `Type`,
            headerClassName: `w-1/10 text-center ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                return (
                    <span className="mx-auto" data-testid="cve-type">
                        <CveType types={original.vulnerabilityTypes} />
                    </span>
                );
            },
            id: cveSortFields.CVE_TYPE,
            accessor: 'vulnerabilityTypes',
            sortField: cveSortFields.CVE_TYPE,
            sortable: true,
        },
        {
            Header: `Fixable`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                return (
                    <VulnerabilityFixableIconText isFixable={original.isFixable} isTextOnly={pdf} />
                );
            },
            id: cveSortFields.FIXABLE,
            accessor: 'isFixable',
            sortField: cveSortFields.FIXABLE,
            sortable: false,
        },
        {
            Header: `Active`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                return original.activeState?.state || 'Undetermined';
            },
            id: cveSortFields.ACTIVE,
            accessor: 'isActive',
            sortField: cveSortFields.ACTIVE,
            sortable: false,
        },
        {
            Header: `Fixed in`,
            headerClassName: `w-1/12 ${defaultHeaderClassName}`,
            className: `w-1/12 word-break-all ${defaultColumnClassName}`,
            Cell: ({ original }) => original.fixedByVersion || '-',
            id: cveSortFields.FIXEDIN,
            accessor: 'fixedByVersion',
            sortField: cveSortFields.FIXEDIN,
        },
        {
            Header: `Severity`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 text-center ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => {
                return (
                    <VulnerabilitySeverityIconText severity={original.severity} isTextOnly={pdf} />
                );
            },
            id: cveSortFields.SEVERITY,
            accessor: 'severity',
            sortField: cveSortFields.SEVERITY,
        },
        {
            Header: `CVSS Score`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { cvss, scoreVersion } = original;
                return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
            },
            id: cveSortFields.CVSS_SCORE,
            accessor: 'cvss',
            sortField: cveSortFields.CVSS_SCORE,
        },
        {
            Header: `Env. Impact`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { envImpact } = original;
                // eslint-disable-next-line eqeqeq
                return envImpact == Number(envImpact)
                    ? `${(envImpact * 100).toFixed(0)}% affected`
                    : '-';
            },
            id: cveSortFields.ENV_IMPACT,
            accessor: 'envImpact',
            sortField: cveSortFields.ENV_IMPACT,
            sortable: false,
        },
        {
            Header: `Impact Score`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const { impactScore } = original;
                // eslint-disable-next-line eqeqeq
                return impactScore == Number(impactScore) ? impactScore.toFixed(1) : '-';
            },
            id: cveSortFields.IMPACT_SCORE,
            accessor: 'impactScore',
            sortField: cveSortFields.IMPACT_SCORE,
        },
        {
            Header: `Entities`,
            headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <TableCountLinks row={original} textOnly={inFindingsSection || pdf} />
            ),
            accessor: 'entities',
            sortable: false,
        },
        {
            Header: `Discovered Time`,
            headerClassName: `w-1/10 text-left ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => <DateTimeField date={original.createdAt} asString={pdf} />,
            id: cveSortFields.CVE_CREATED_TIME,
            accessor: 'createdAt',
            sortField: cveSortFields.CVE_CREATED_TIME,
        },
        {
            Header: `Discovered in Image`,
            headerClassName: `w-1/9 text-left ${nonSortableHeaderClassName}`,
            className: `w-1/9 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <DateTimeField date={original.discoveredAtImage} asString={pdf} />
            ),
            id: cveSortFields.CVE_DISCOVERED_AT_IMAGE_TIME,
            accessor: 'discoveredAtImage',
            sortable: false,
        },
        {
            Header: `Published`,
            headerClassName: `w-1/12 ${defaultHeaderClassName}`,
            className: `w-1/12 ${defaultColumnClassName}`,
            Cell: ({ original, pdf }) => (
                <DateTimeField date={original.publishedOn} asString={pdf} />
            ),
            id: cveSortFields.PUBLISHED,
            accessor: 'published',
            sortField: cveSortFields.PUBLISHED,
        },
    ];

    if (currentEntityType === entityTypes.NODE_CVE || currentEntityType === entityTypes.IMAGE_CVE) {
        tableColumns.splice(3, 0, {
            Header: `Operating System`,
            headerClassName: `w-1/10 ${defaultHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            id: cveSortFields.OPERATING_SYSTEM,
            accessor: 'operatingSystem',
            sortField: cveSortFields.OPERATING_SYSTEM,
        });
    }

    const nonNullTableColumns = tableColumns.filter((col) => col);

    const cveColumnsBasedOnContext = getFilteredCVEColumns(
        nonNullTableColumns,
        workflowState,
        isFeatureFlagEnabled
    );

    return removeEntityContextColumns(cveColumnsBasedOnContext, workflowState);
}

export function renderCveDescription(row) {
    const { original } = row;
    return (
        <div
            className="pointer-events-none bottom-0 absolute px-2 pb-3 pt-1 flex h-12 items-center"
            data-testid="subcomponent-row"
        >
            <div className="line-clamp leading-normal" data-testid="cve-description">
                {original.summary || 'No description available.'}
            </div>
        </div>
    );
}

const VulnMgmtCves = ({
    selectedRowId,
    search,
    sort,
    page,
    data,
    totalResults,
    refreshTrigger,
}) => {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();

    // Although request requires only WorkflowAdministration,
    // also require require resources for Policies route.
    const hasWriteAccessForAddToPolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');

    const [selectedCveIds, setSelectedCveIds] = useState([]);
    const [bulkActionCveIds, setBulkActionCveIds] = useState([]);

    const workflowState = useContext(workflowStateContext);

    const cveType = workflowState.getCurrentEntityType();

    let cveQuery = '';

    switch (cveType) {
        case entityTypes.NODE_CVE: {
            cveQuery = gql`
                query getNodeCves($query: String, $scopeQuery: String, $pagination: Pagination) {
                    results: nodeVulnerabilities(query: $query, pagination: $pagination) {
                        ...nodeCVEFields
                    }
                    count: nodeVulnerabilityCount(query: $query)
                }
                ${NODE_CVE_LIST_FRAGMENT}
            `;
            break;
        }
        case entityTypes.CLUSTER_CVE: {
            cveQuery = gql`
                query getClusterCves($query: String, $scopeQuery: String, $pagination: Pagination) {
                    results: clusterVulnerabilities(query: $query, pagination: $pagination) {
                        ...clusterCVEFields
                    }
                    count: clusterVulnerabilityCount(query: $query)
                }
                ${CLUSTER_CVE_LIST_FRAGMENT}
            `;
            break;
        }
        case entityTypes.IMAGE_CVE:
        default: {
            cveQuery = gql`
                query getImageCves($query: String, $scopeQuery: String, $pagination: Pagination) {
                    results: imageVulnerabilities(query: $query, pagination: $pagination) {
                        ...imageCVEFields
                    }
                    count: imageVulnerabilityCount(query: $query)
                }
                ${IMAGE_CVE_LIST_FRAGMENT}
            `;
            break;
        }
    }

    const tableSort = sort || defaultCveSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause({
                ...search,
                cachebuster: refreshTrigger,
            }),
            scopeQuery: '',
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    const addToPolicy = (cve) => (e) => {
        e.stopPropagation();

        const cveIdsToAdd = cve ? [cve] : selectedCveIds;

        if (cveIdsToAdd.length) {
            setBulkActionCveIds(cveIdsToAdd);
        } else {
            throw new Error(
                'Logic error: tried to open Add to Policy dialog without any policy selected.'
            );
        }
    };

    function closeDialog(idsToStaySelected = []) {
        setBulkActionCveIds([]);
        setSelectedCveIds(idsToStaySelected);
    }

    const renderRowActionButtons =
        hasWriteAccessForAddToPolicy
            ? ({ cve }) => (
                  <div className="flex border-2 border-r-2 border-base-400 bg-base-100">
                      {hasWriteAccessForAddToPolicy && cveType === entityTypes.IMAGE_CVE && (
                          <RowActionButton
                              text="Add to policy"
                              onClick={addToPolicy(cve)}
                              icon={<Plus className="my-1 h-4 w-4" />}
                          />
                      )}
                  </div>
              )
            : null;

    const tableHeaderComponents = (
        <>
            {hasWriteAccessForAddToPolicy && cveType === entityTypes.IMAGE_CVE && (
                <PanelButton
                    icon={<Plus className="h-4 w-4" />}
                    className="btn-icon btn-tertiary"
                    onClick={addToPolicy()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Add Selected CVEs to Policy"
                >
                    Add to policy
                </PanelButton>
            )}
        </>
    );

    return (
        <>
            <WorkflowListPage
                data={data}
                totalResults={totalResults}
                query={cveQuery}
                queryOptions={queryOptions}
                idAttribute="id"
                entityListType={cveType}
                getTableColumns={getCveTableColumns}
                selectedRowId={selectedRowId}
                search={search}
                sort={tableSort}
                page={page}
                showSubrows
                SubComponent={renderCveDescription}
                checkbox
                tableHeaderComponents={tableHeaderComponents}
                selection={selectedCveIds}
                setSelection={setSelectedCveIds}
                renderRowActionButtons={renderRowActionButtons}
            />
            {bulkActionCveIds.length > 0 && (
                <CveBulkActionDialogue
                    closeAction={closeDialog}
                    bulkActionCveIds={bulkActionCveIds}
                    cveType={cveType}
                />
            )}
        </>
    );
};

VulnMgmtCves.propTypes = {
    ...workflowListPropTypes,
    refreshTrigger: PropTypes.number,
};
VulnMgmtCves.defaultProps = {
    ...workflowListDefaultProps,
    refreshTrigger: 0,
};

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(VulnMgmtCves);

import React, { useContext, useState } from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { withRouter } from 'react-router-dom';

import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import RowActionMenu from 'Components/RowActionMenu';
import DateTimeField from 'Components/DateTimeField';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import Menu from 'Components/Menu';
import TableCountLinks from 'Components/workflow/TableCountLinks';
import TopCvssLabel from 'Components/TopCvssLabel';
import PanelButton from 'Components/PanelButton';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import useAnalytics, { GLOBAL_SNOOZE_CVE } from 'hooks/useAnalytics';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';
import { actions as notificationActions } from 'reducers/notifications';
import { suppressVulns, unsuppressVulns } from 'services/VulnerabilitiesService';
import queryService from 'utils/queryService';
import removeEntityContextColumns from 'utils/tableUtils';
import { getViewStateFromSearch } from 'utils/searchUtils';
import { cveSortFields } from 'constants/sortFields';
import { snoozeDurations, durations } from 'constants/timeWindows';
import {
    IMAGE_CVE_LIST_FRAGMENT,
    NODE_CVE_LIST_FRAGMENT,
    CLUSTER_CVE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';

import CveType from 'Components/CveType';

import CveBulkActionDialogue from './CveBulkActionDialogue';

import { entityCountNounOrdinaryCase } from '../../entitiesForVulnerabilityManagement';
import WorkflowListPage from '../WorkflowListPage';
import { getFilteredCVEColumns, parseCveNamesFromIds } from './ListCVEs.utils';

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
    history,
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
    const { analyticsTrack } = useAnalytics();
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadWriteAccess } = usePermissions();

    // Although request requires only WorkflowAdministration,
    // also require require resources for Policies route.
    const hasWriteAccessForAddToPolicy =
        hasReadWriteAccess('WorkflowAdministration') && isRouteEnabled('policy-management');

    // Forbidden failures are explicit for Approvals and Requests but only implicit for Image.
    const hasWriteAccessForRiskAcceptance =
        hasReadWriteAccess('Image') &&
        hasReadWriteAccess('VulnerabilityManagementApprovals') &&
        hasReadWriteAccess('VulnerabilityManagementRequests');

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');

    const [selectedCveIds, setSelectedCveIds] = useState([]);
    const [bulkActionCveIds, setBulkActionCveIds] = useState([]);

    const workflowState = useContext(workflowStateContext);

    const cveType = workflowState.getCurrentEntityType();

    const shouldRenderGlobalSnooze = !isUnifiedDeferralEnabled || cveType !== entityTypes.IMAGE_CVE;

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

    const viewingSuppressed = getViewStateFromSearch(search, cveSortFields.SUPPRESSED);

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

    function trackGlobalSnooze(cveNames, entityType, duration) {
        let type = undefined;

        if (entityType === resourceTypes.NODE_CVE) {
            type = 'NODE';
        } else if (entityType === resourceTypes.CLUSTER_CVE) {
            type = 'PLATFORM';
        } else {
            // The entity type is IMAGE_CVE or something unexpected, so we don't want to track it
            return;
        }

        cveNames.forEach((cve) => {
            analyticsTrack({
                event: GLOBAL_SNOOZE_CVE,
                properties: { type, cve, duration },
            });
        });
    }

    const suppressCves = (cve, duration) => (e) => {
        e.stopPropagation();

        const currentEntityType = workflowState.getCurrentEntity().entityType;
        const cveIdsToToggle = cve ? [cve] : selectedCveIds;

        const selectedCveNames = parseCveNamesFromIds(cveIdsToToggle);

        suppressVulns(cveType, selectedCveNames, duration)
            .then(() => {
                setSelectedCveIds([]);

                // changing this param value on the query vars, to force the query to refetch
                setRefreshTrigger(Math.random());

                addToast(
                    `Successfully deferred and approved ${entityCountNounOrdinaryCase(
                        selectedCveNames.length,
                        currentEntityType
                    )} globally`
                );
                setTimeout(removeToast, 2000);

                trackGlobalSnooze(selectedCveNames, currentEntityType, duration);
            })
            .catch((evt) => {
                addToast(`Could not defer and approve all of the selected CVEs: ${evt.message}`);
                setTimeout(removeToast, 2000);
            });
    };

    const unsuppressCves = (cve) => (e) => {
        e.stopPropagation();

        const currentEntityType = workflowState.getCurrentEntity().entityType;
        const cveIdsToToggle = cve ? [cve] : selectedCveIds;

        const selectedCveNames = parseCveNamesFromIds(cveIdsToToggle);

        unsuppressVulns(cveType, selectedCveNames)
            .then(() => {
                setSelectedCveIds([]);

                // changing this param value on the query vars, to force the query to refetch
                setRefreshTrigger(Math.random());

                addToast(
                    `Successfully reobserved ${entityCountNounOrdinaryCase(
                        selectedCveNames.length,
                        currentEntityType
                    )} globally`
                );
                setTimeout(removeToast, 2000);
            })
            .catch((evt) => {
                addToast(`Could not reobserve all of the selected CVEs: ${evt.message}`);
                setTimeout(removeToast, 2000);
            });
    };

    const toggleSuppressedView = () => {
        const currentSearchState = workflowState.getCurrentSearchState();

        const targetSearchState = { ...currentSearchState };
        if (viewingSuppressed) {
            targetSearchState[cveSortFields.SUPPRESSED] = false;
        } else {
            targetSearchState[cveSortFields.SUPPRESSED] = true;
        }

        const newWorkflowState = workflowState.setSearch(targetSearchState);
        const newUrl = newWorkflowState.toUrl();
        history.push(newUrl);
    };

    function closeDialog(idsToStaySelected = []) {
        setBulkActionCveIds([]);
        setSelectedCveIds(idsToStaySelected);
    }

    const snoozeOptions = (cve) => {
        return Object.keys(snoozeDurations).map((d) => {
            return { label: snoozeDurations[d], onClick: suppressCves(cve, durations[d]) };
        });
    };

    const renderRowActionButtons =
        hasWriteAccessForAddToPolicy || hasWriteAccessForRiskAcceptance
            ? ({ cve }) => (
                  <div className="flex border-2 border-r-2 border-base-400 bg-base-100">
                      {hasWriteAccessForAddToPolicy && cveType === entityTypes.IMAGE_CVE && (
                          <RowActionButton
                              text="Add to policy"
                              onClick={addToPolicy(cve)}
                              icon={<Icon.Plus className="my-1 h-4 w-4" />}
                          />
                      )}
                      {hasWriteAccessForRiskAcceptance &&
                          !viewingSuppressed &&
                          shouldRenderGlobalSnooze && (
                              <RowActionMenu
                                  className="h-full min-w-30"
                                  border="border-l-2 border-base-400"
                                  icon={<Icon.BellOff className="h-4 w-4" />}
                                  options={snoozeOptions(cve)}
                                  text="Defer and approve CVE"
                              />
                          )}
                      {hasWriteAccessForRiskAcceptance &&
                          viewingSuppressed &&
                          shouldRenderGlobalSnooze && (
                              <RowActionButton
                                  text="Reobserve CVE"
                                  border="border-l-2 border-base-400"
                                  onClick={unsuppressCves(cve)}
                                  icon={<Icon.Bell className="my-1 h-4 w-4" />}
                              />
                          )}
                  </div>
              )
            : null;

    const viewButtonText = viewingSuppressed ? 'View observed' : 'View deferred';

    const tableHeaderComponents = (
        <>
            {hasWriteAccessForAddToPolicy && cveType === entityTypes.IMAGE_CVE && (
                <PanelButton
                    icon={<Icon.Plus className="h-4 w-4" />}
                    className="btn-icon btn-tertiary"
                    onClick={addToPolicy()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Add Selected CVEs to Policy"
                >
                    Add to policy
                </PanelButton>
            )}
            {hasWriteAccessForRiskAcceptance && !viewingSuppressed && shouldRenderGlobalSnooze && (
                <Menu
                    className="h-full min-w-30 ml-2"
                    menuClassName="bg-base-100 min-w-28"
                    buttonClass="btn-icon btn-tertiary"
                    buttonText="Defer and approve"
                    buttonIcon={<Icon.BellOff className="h-4 w-4 mr-2" />}
                    options={snoozeOptions()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Defer and approve selected CVEs"
                />
            )}

            {hasWriteAccessForRiskAcceptance && viewingSuppressed && shouldRenderGlobalSnooze && (
                <PanelButton
                    icon={<Icon.Bell className="h-4 w-4" />}
                    className="btn-icon btn-tertiary ml-2"
                    onClick={unsuppressCves()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Reobserve selected CVEs"
                >
                    Reobserve
                </PanelButton>
            )}

            <span className="w-px bg-base-400 ml-2" />
            {shouldRenderGlobalSnooze && (
                <PanelButton
                    icon={
                        viewingSuppressed ? (
                            <Icon.Zap className="h-4 w-4" />
                        ) : (
                            <Icon.Archive className="h-4 w-4" />
                        )
                    }
                    className="btn-icon btn-tertiary ml-2"
                    onClick={toggleSuppressedView}
                    tooltip={`${viewButtonText} CVEs`}
                >
                    {viewButtonText}
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
    setRefreshTrigger: PropTypes.func,
};
VulnMgmtCves.defaultProps = {
    ...workflowListDefaultProps,
    refreshTrigger: 0,
    setRefreshTrigger: null,
};

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default withRouter(connect(null, mapDispatchToProps)(VulnMgmtCves));

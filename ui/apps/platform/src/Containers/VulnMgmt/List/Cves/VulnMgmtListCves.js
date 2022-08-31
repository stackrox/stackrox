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
import LabelChip from 'Components/LabelChip';
import Menu from 'Components/Menu';
import TableCountLinks from 'Components/workflow/TableCountLinks';
import TopCvssLabel from 'Components/TopCvssLabel';
import PanelButton from 'Components/PanelButton';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import workflowStateContext from 'Containers/workflowStateContext';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import queryService from 'utils/queryService';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import { actions as notificationActions } from 'reducers/notifications';
import { suppressVulns, unsuppressVulns } from 'services/VulnerabilitiesService';
import removeEntityContextColumns from 'utils/tableUtils';
import { getViewStateFromSearch } from 'utils/searchUtils';
import { cveSortFields } from 'constants/sortFields';
import { snoozeDurations, durations } from 'constants/timeWindows';
import {
    VULN_CVE_LIST_FRAGMENT,
    IMAGE_CVE_LIST_FRAGMENT,
    NODE_CVE_LIST_FRAGMENT,
    CLUSTER_CVE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import useFeatureFlags from 'hooks/useFeatureFlags';

import CVSSSeverityLabel from 'Components/CVSSSeverityLabel';
import CveType from 'Components/CveType';
import CveBulkActionDialogue from './CveBulkActionDialogue';

import { getFilteredCVEColumns } from './ListCVEs.utils';
import { resourceLabels } from '../../../../messages/common';

export const defaultCveSort = [
    {
        id: cveSortFields.CVSS_SCORE,
        desc: true,
    },
];

export function getCveTableColumns(workflowState) {
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
            headerClassName: `w-1/10 text-center ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            Cell: ({ original }) => {
                const fixableFlag = original.isFixable ? (
                    <LabelChip text="Fixable" type="success" size="large" />
                ) : (
                    'No'
                );
                return <div className="mx-auto">{fixableFlag}</div>;
            },
            id: cveSortFields.FIXABLE,
            accessor: 'isFixable',
            sortField: cveSortFields.FIXABLE,
            sortable: false,
        },
        {
            Header: `Active`,
            headerClassName: `w-1/10 text-center ${nonSortableHeaderClassName}`,
            className: `w-1/10 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const activeStatus = original.activeState?.state || 'Undetermined';
                switch (activeStatus) {
                    case 'Active': {
                        return (
                            <div className="mx-auto">
                                <LabelChip text={activeStatus} type="alert" size="large" />
                            </div>
                        );
                    }
                    case 'Inactive': {
                        return <div className="mx-auto">{activeStatus}</div>;
                    }
                    case 'Undetermined':
                    default: {
                        return <div className="mx-auto">Undetermined</div>;
                    }
                }
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
            Cell: ({ original }) => {
                return <CVSSSeverityLabel severity={original.severity} />;
            },
            id: cveSortFields.SEVERITY,
            accessor: 'severity',
            sortField: cveSortFields.SEVERITY,
        },
        {
            Header: `CVSS Score`,
            headerClassName: `w-1/10 text-center ${defaultHeaderClassName}`,
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

    const cveColumnsBasedOnContext = getFilteredCVEColumns(nonNullTableColumns, workflowState);

    return removeEntityContextColumns(cveColumnsBasedOnContext, workflowState);
}

export function renderCveDescription(row) {
    const { original } = row;
    return (
        <div
            className="pointer-events-none bottom-0 absolute px-2 pb-3 pt-1 text-base-500 flex h-12 items-center"
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
    const [selectedCveIds, setSelectedCveIds] = useState([]);
    const [bulkActionCveIds, setBulkActionCveIds] = useState([]);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');
    const usesPostgres = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

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
        case entityTypes.IMAGE_CVE: {
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
        // TODO: remove the deprecated one-CVE-to-rule-them-all type, and move default case to IMAGE_CVE
        case entityTypes.CVE:
        default: {
            cveQuery = gql`
                query getCves($query: String, $scopeQuery: String, $pagination: Pagination) {
                    results: vulnerabilities(query: $query, pagination: $pagination) {
                        ...cveFields
                    }
                    count: vulnerabilityCount(query: $query)
                }
                ${VULN_CVE_LIST_FRAGMENT}
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

    const suppressCves = (cve, duration) => (e) => {
        e.stopPropagation();

        const currentEntityType = workflowState.getCurrentEntity().entityType;
        const entityTypeDisplayName = resourceLabels[currentEntityType];
        const cvesToToggle = cve ? [cve] : selectedCveIds;
        suppressVulns(cveType, cvesToToggle, duration)
            .then(() => {
                setSelectedCveIds([]);

                // changing this param value on the query vars, to force the query to refetch
                setRefreshTrigger(Math.random());

                // can't use pluralize() because of this bug: https://github.com/blakeembrey/pluralize/issues/127
                const pluralizedCVEs =
                    cvesToToggle.length === 1 ? entityTypeDisplayName : `${entityTypeDisplayName}s`;

                addToast(
                    `Successfully deferred and approved ${cvesToToggle.length} ${pluralizedCVEs} globally`
                );
                setTimeout(removeToast, 2000);
            })
            .catch((evt) => {
                addToast(`Could not defer and approve all of the selected CVEs: ${evt.message}`);
                setTimeout(removeToast, 2000);
            });
    };

    const unsuppressCves = (cve) => (e) => {
        e.stopPropagation();

        const currentEntityType = workflowState.getCurrentEntity().entityType;
        const entityTypeDisplayName = resourceLabels[currentEntityType];
        const cveIdsToToggle = cve ? [cve] : selectedCveIds;
        unsuppressVulns(cveType, cveIdsToToggle)
            .then(() => {
                setSelectedCveIds([]);

                // changing this param value on the query vars, to force the query to refetch
                setRefreshTrigger(Math.random());

                // can't use pluralize() because of this bug: https://github.com/blakeembrey/pluralize/issues/127
                const pluralizedCVEs =
                    cveIdsToToggle.length === 1
                        ? entityTypeDisplayName
                        : `${entityTypeDisplayName}s`;

                addToast(
                    `Successfully reobserved ${cveIdsToToggle.length} ${pluralizedCVEs} globally`
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

    const renderRowActionButtons = ({ cve }) => (
        <div className="flex border-2 border-r-2 border-base-400 bg-base-100">
            <RowActionButton
                text="Add to Policy"
                onClick={addToPolicy(cve)}
                date-testid="row-action-add-to-policy"
                icon={<Icon.Plus className="my-1 h-4 w-4" />}
            />
            {!viewingSuppressed && (
                <RowActionMenu
                    className="h-full min-w-30"
                    border="border-l-2 border-base-400"
                    icon={<Icon.BellOff className="h-4 w-4" />}
                    options={snoozeOptions(cve)}
                    text="Defer and Approve CVE"
                    dataTestId="row-action-suppress"
                />
            )}
            {viewingSuppressed && (
                <RowActionButton
                    text="Reobserve CVE"
                    border="border-l-2 border-base-400"
                    onClick={unsuppressCves(cve)}
                    date-testid="row-action-unsuppress"
                    icon={<Icon.Bell className="my-1 h-4 w-4" />}
                    dataTestId="row-action-unsuppress"
                />
            )}
        </div>
    );

    const viewButtonText = viewingSuppressed ? 'View Observed' : 'View Deferred';

    const tableHeaderComponents = (
        <>
            <PanelButton
                icon={<Icon.Plus className="h-4 w-4" />}
                className="btn-icon btn-tertiary"
                onClick={addToPolicy()}
                disabled={selectedCveIds.length === 0}
                tooltip="Add Selected CVEs to Policy"
                dataTestId="panel-button-add-cves-to-policy"
            >
                Add to Policy
            </PanelButton>
            {!viewingSuppressed && (
                <Menu
                    className="h-full min-w-30 ml-2"
                    menuClassName="bg-base-100 min-w-28"
                    buttonClass="btn-icon btn-tertiary"
                    buttonText="Defer and Approve"
                    buttonIcon={<Icon.BellOff className="h-4 w-4 mr-2" />}
                    options={snoozeOptions()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Defer and Approve Selected CVEs"
                    dataTestId="panel-button-suppress-selected-cves"
                />
            )}

            {viewingSuppressed && (
                <PanelButton
                    icon={<Icon.Bell className="h-4 w-4" />}
                    className="btn-icon btn-tertiary ml-2"
                    onClick={unsuppressCves()}
                    disabled={selectedCveIds.length === 0}
                    tooltip="Reobserve Selected CVEs"
                    dataTestId="panel-button-unsuppress-selected-cves"
                >
                    Reobserve
                </PanelButton>
            )}

            <span className="w-px bg-base-400 ml-2" />
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
                dataTestId="panel-button-toggle-suppressed-cves-view"
            >
                {viewButtonText}
            </PanelButton>
        </>
    );

    return (
        <>
            <WorkflowListPage
                data={data}
                totalResults={totalResults}
                query={cveQuery}
                queryOptions={queryOptions}
                idAttribute={showVMUpdates && usesPostgres ? 'id' : 'cve'}
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

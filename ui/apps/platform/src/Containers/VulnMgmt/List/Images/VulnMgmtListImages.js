import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';
import { List } from 'react-feather';

import PanelButton from 'Components/PanelButton';
import TopCvssLabel from 'Components/TopCvssLabel';
import ImageTableCountLinks from 'Components/workflow/ImageTableCountLinks';
import StatusChip from 'Components/StatusChip';
import CVEStackedPill from 'Components/CVEStackedPill';
import DateTimeField from 'Components/DateTimeField';
import {
    defaultHeaderClassName,
    nonSortableHeaderClassName,
    defaultColumnClassName,
} from 'Components/Table';
import entityTypes from 'constants/entityTypes';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import WorkflowListPage from 'Containers/Workflow/WorkflowListPage';
import workflowStateContext from 'Containers/workflowStateContext';
import { imageWatchStatuses } from 'Containers/VulnMgmt/VulnMgmt.constants';
import {
    IMAGE_LIST_FRAGMENT,
    OLD_IMAGE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import getImageScanMessage from 'Containers/VulnMgmt/VulnMgmt.utils/getImageScanMessage';
import { workflowListPropTypes, workflowListDefaultProps } from 'constants/entityPageProps';
import removeEntityContextColumns from 'utils/tableUtils';
import { imageSortFields } from 'constants/sortFields';
import queryService from 'utils/queryService';
import useFeatureFlags from 'hooks/useFeatureFlags';
import WatchedImagesDialog from './WatchedImagesDialog';

export const defaultImageSort = [
    {
        id: imageSortFields.PRIORITY,
        desc: false,
    },
];

export function getCurriedImageTableColumns(watchedImagesTrigger, isFeatureFlagEnabled) {
    const isFrontendVMUpdatesEnabled = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    return function getImageTableColumns(workflowState) {
        const tableColumns = [
            {
                Header: 'Id',
                headerClassName: 'hidden',
                className: 'hidden',
                accessor: 'id',
            },
            {
                Header: `Image`,
                headerClassName: `w-1/6 ${defaultHeaderClassName}`,
                className: `w-1/6 word-break-all ${defaultColumnClassName}`,
                id: imageSortFields.NAME,
                accessor: 'name.fullName',
                sortField: imageSortFields.NAME,
            },
            {
                Header: isFrontendVMUpdatesEnabled ? 'Image CVEs' : 'CVEs',
                entityType: isFrontendVMUpdatesEnabled ? entityTypes.IMAGE_CVE : entityTypes.CVE,
                headerClassName: `w-1/6 ${defaultHeaderClassName}`,
                className: `w-1/6 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { vulnCounter, id, scanTime, scanNotes, notes } = original;

                    const newState = workflowState
                        .pushListItem(id)
                        .pushList(
                            isFrontendVMUpdatesEnabled ? entityTypes.IMAGE_CVE : entityTypes.CVE
                        );
                    const url = newState.toUrl();
                    const fixableUrl = newState.setSearch({ Fixable: true }).toUrl();

                    return (
                        <CVEStackedPill
                            vulnCounter={vulnCounter}
                            url={url}
                            fixableUrl={fixableUrl}
                            entityName="Image"
                            hideLink={pdf}
                            scanTime={scanTime}
                            scanMessage={getImageScanMessage(notes || [], scanNotes || [])}
                        />
                    );
                },
                id: imageSortFields.CVE_COUNT,
                accessor: 'vulnCounter.all.total',
                sortField: imageSortFields.CVE_COUNT,
            },
            {
                Header: `Top CVSS`,
                headerClassName: `w-1/12 text-center ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { topVuln } = original;
                    if (!topVuln) {
                        return (
                            <div className="mx-auto flex flex-col">
                                <span>–</span>
                            </div>
                        );
                    }
                    const { cvss, scoreVersion } = topVuln;
                    return <TopCvssLabel cvss={cvss} version={scoreVersion} />;
                },
                id: imageSortFields.TOP_CVSS,
                accessor: 'topVuln.cvss',
                sortField: imageSortFields.TOP_CVSS,
            },
            {
                Header: `Created`,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { metadata } = original;
                    if (!metadata || !metadata.v1) {
                        return '–';
                    }
                    return <DateTimeField date={metadata.v1.created} asString={pdf} />;
                },
                id: imageSortFields.CREATED_TIME,
                accessor: 'metadata.v1.created',
                sortField: imageSortFields.CREATED_TIME,
            },
            {
                Header: `Scan Time`,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => {
                    const { scanTime } = original;
                    if (!scanTime) {
                        return '–';
                    }
                    return <DateTimeField date={scanTime} asString={pdf} />;
                },
                id: imageSortFields.SCAN_TIME,
                accessor: 'scanTime',
                sortField: imageSortFields.SCAN_TIME,
            },
            {
                Header: `Image OS`,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                Cell: ({ original }) => {
                    const { operatingSystem } = original;
                    if (!operatingSystem) {
                        return '–';
                    }
                    return <span>{operatingSystem}</span>;
                },
                id: imageSortFields.IMAGE_OS,
                accessor: 'operatingSystem',
                sortField: imageSortFields.IMAGE_OS,
            },
            {
                Header: 'Image Status',
                headerClassName: `w-1/10 ${nonSortableHeaderClassName}`,
                className: `w-1/10 ${defaultColumnClassName} content-center`,
                Cell: ({ original, pdf }) => {
                    const { deploymentCount, watchStatus } = original;
                    const imageStatus = deploymentCount === 0 ? 'inactive' : 'active';
                    const isWatched = watchStatus === imageWatchStatuses.WATCHED;
                    return (
                        <div className="flex-col justify-center items-center w-full">
                            <StatusChip status={imageStatus} asString={pdf} />
                            {isWatched && (
                                <button
                                    type="button"
                                    onClick={watchedImagesTrigger}
                                    className="text-primary-700 text-center underline w-full"
                                >
                                    Scanning via watch tag
                                </button>
                            )}
                        </div>
                    );
                },
                id: imageSortFields.IMAGE_STATUS,
                accessor: 'deploymentCount',
                sortField: imageSortFields.IMAGE_STATUS,
                sortable: false,
            },
            {
                Header: `Entities`,
                entityType: entityTypes.DEPLOYMENT,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                Cell: ({ original, pdf }) => (
                    <ImageTableCountLinks
                        row={original}
                        textOnly={pdf}
                        isFrontendVMUpdatesEnabled={isFrontendVMUpdatesEnabled}
                    />
                ),
                accessor: 'entities',
                sortable: false,
            },
            {
                Header: `Risk Priority`,
                headerClassName: `w-1/12 ${defaultHeaderClassName}`,
                className: `w-1/12 ${defaultColumnClassName}`,
                id: imageSortFields.PRIORITY,
                accessor: 'priority',
                sortField: imageSortFields.PRIORITY,
            },
        ];
        return removeEntityContextColumns(tableColumns, workflowState);
    };
}

const VulnMgmtImages = ({
    selectedRowId,
    search,
    sort,
    page,
    data,
    totalResults,
    refreshTrigger,
    setRefreshTrigger,
}) => {
    const [showWatchedImagesDialog, setShowWatchedImagesDialog] = useState(false);
    const workflowState = useContext(workflowStateContext);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVmUpdates = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');
    const fragmentToUse = showVmUpdates ? IMAGE_LIST_FRAGMENT : OLD_IMAGE_LIST_FRAGMENT;

    const inactiveImageScanningEnabled = workflowState.isBaseList(entityTypes.IMAGE);

    const query = gql`
        query getImages($query: String, $pagination: Pagination) {
            results: images(query: $query, pagination: $pagination) {
                ...imageFields
            }
            count: imageCount(query: $query)
        }
        ${fragmentToUse}
    `;

    const tableSort = sort || defaultImageSort;
    const queryOptions = {
        variables: {
            query: queryService.objectToWhereClause({
                ...search,
                cachebuster: refreshTrigger,
            }),
            pagination: queryService.getPagination(tableSort, page, LIST_PAGE_SIZE),
        },
    };

    function toggleWatchedImagesDialog(e) {
        e.stopPropagation();

        if (showWatchedImagesDialog) {
            // changing this param value on the query vars, to force the query to refetch
            setRefreshTrigger(Math.random());
        }

        setShowWatchedImagesDialog(!showWatchedImagesDialog);
    }
    const tableHeaderComponents = inactiveImageScanningEnabled ? (
        <PanelButton
            icon={<List className="h-4 w-4" />}
            className="btn-icon btn-tertiary"
            onClick={toggleWatchedImagesDialog}
            tooltip="Manage Scanning of Watched Images"
            dataTestId="panel-button-manage-inactive-images"
        >
            Manage Watches
        </PanelButton>
    ) : null;

    const getImageTableColumns = getCurriedImageTableColumns(
        toggleWatchedImagesDialog,
        isFeatureFlagEnabled
    );

    return (
        <>
            <WorkflowListPage
                data={data}
                totalResults={totalResults}
                query={query}
                queryOptions={queryOptions}
                entityListType={entityTypes.IMAGE}
                getTableColumns={getImageTableColumns}
                selectedRowId={selectedRowId}
                search={search}
                sort={tableSort}
                page={page}
                tableHeaderComponents={tableHeaderComponents}
            />
            {showWatchedImagesDialog && (
                <WatchedImagesDialog closeDialog={toggleWatchedImagesDialog} />
            )}
        </>
    );
};

VulnMgmtImages.propTypes = {
    ...workflowListPropTypes,
    refreshTrigger: PropTypes.number,
    setRefreshTrigger: PropTypes.func,
};

VulnMgmtImages.defaultProps = {
    ...workflowListDefaultProps,
    refreshTrigger: 0,
    setRefreshTrigger: null,
};

export default VulnMgmtImages;

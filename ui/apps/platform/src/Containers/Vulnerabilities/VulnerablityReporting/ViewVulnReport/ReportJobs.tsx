import React, { useState } from 'react';
import {
    ActionsColumn,
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import {
    Alert,
    AlertGroup,
    AlertVariant,
    Bullseye,
    Button,
    Card,
    Divider,
    Flex,
    FlexItem,
    Pagination,
    SelectOption,
    Spinner,
    Switch,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, FilterIcon } from '@patternfly/react-icons';

import { ReportConfiguration, RunState, runStates } from 'services/ReportsService.types';
import { getDateTime } from 'utils/dateUtils';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import useInterval from 'hooks/useInterval';
import useFetchReportHistory from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReportHistory';
import { getRequestQueryString } from 'Containers/Vulnerabilities/VulnerablityReporting/api/apiUtils';
import useURLSort from 'hooks/useURLSort';
import { saveFile } from 'services/DownloadService';
import useDeleteDownloadModal from 'Containers/Vulnerabilities/VulnerablityReporting/hooks/useDeleteDownloadModal';
import useAuthStatus from 'hooks/useAuthStatus';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import ReportJobStatus from './ReportJobStatus';
import JobDetails from './JobDetails';
import DeleteModal from '../components/DeleteModal';

export type RunHistoryProps = {
    reportId: string;
};

const sortOptions = {
    sortFields: ['Report Completion Time'],
    defaultSortOption: { field: 'Report Completion Time', direction: 'desc' } as const,
};

function ReportJobs({ reportId }: RunHistoryProps) {
    const { currentUser } = useAuthStatus();
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const [filteredStatuses, setFilteredStatuses] = useState<RunState[]>([]);
    const [showOnlyMyJobs, setShowOnlyMyJobs] = React.useState<boolean>(false);
    const expandedRowSet = useSet<string>();

    const query = getRequestQueryString({
        'Report state': filteredStatuses,
    });

    const { reportSnapshots, isLoading, error, fetchReportSnapshots } = useFetchReportHistory({
        id: reportId,
        query,
        page,
        perPage,
        sortOption,
        showMyHistory: showOnlyMyJobs,
    });

    const {
        openDeleteDownloadModal,
        isDeleteDownloadModalOpen,
        closeDeleteDownloadModal,
        isDeletingDownload,
        onDeleteDownload,
        deleteDownloadError,
    } = useDeleteDownloadModal({
        onCompleted: fetchReportSnapshots,
    });

    const handleChange = (checked: boolean) => {
        setShowOnlyMyJobs(checked);
        setPage(1);
    };

    useInterval(fetchReportSnapshots, 10000);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <CheckboxSelect
                            ariaLabel="CVE severity checkbox select"
                            toggleIcon={<FilterIcon />}
                            selections={filteredStatuses}
                            onChange={(selection) => {
                                // transform the string[] to RunState[]
                                const newRunStates: RunState[] = selection.filter(
                                    (val) => runStates[val] !== undefined
                                ) as RunState[];
                                setFilteredStatuses(newRunStates);
                                setPage(1);
                            }}
                            placeholderText="Filter by status"
                        >
                            <SelectOption value={runStates.PREPARING}>Preparing</SelectOption>
                            <SelectOption value={runStates.WAITING}>Waiting</SelectOption>
                            <SelectOption value={runStates.GENERATED}>
                                Download generated
                            </SelectOption>
                            <SelectOption value={runStates.DELIVERED}>Email delivered</SelectOption>
                            <SelectOption value={runStates.FAILURE}>Error</SelectOption>
                        </CheckboxSelect>
                    </ToolbarItem>
                    <ToolbarItem className="pf-u-flex-grow-1">
                        <Switch
                            id="view-only-my-jobs"
                            label="View only my jobs"
                            labelOff="View only my jobs"
                            isChecked={showOnlyMyJobs}
                            onChange={handleChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            toggleTemplate={({ firstIndex, lastIndex }) => (
                                <span>
                                    <b>
                                        {firstIndex} - {lastIndex}
                                    </b>{' '}
                                    of <b>many</b>
                                </span>
                            )}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            {error && (
                <Bullseye className="pf-u-background-color-100">
                    <EmptyStateTemplate
                        title="Error loading report jobs"
                        headingLevel="h2"
                        icon={ExclamationCircleIcon}
                        iconClassName="pf-u-danger-color-100"
                    >
                        {error}
                    </EmptyStateTemplate>
                </Bullseye>
            )}
            {isLoading && !reportSnapshots && (
                <Bullseye className="pf-u-background-color-100 pf-u-p-lg">
                    <Spinner aria-label="Loading report jobs" />
                </Bullseye>
            )}
            {reportSnapshots && (
                <TableComposable aria-label="Simple table" variant="compact">
                    <Thead>
                        <Tr>
                            <Td>{/* Header for expanded column */}</Td>
                            <Th width={25} sort={getSortParams('Report Completion Time')}>
                                Completed
                            </Th>
                            <Th width={25}>Status</Th>
                            <Th width={50}>Requestor</Th>
                            <Td>{/* Header for table actions column */}</Td>
                        </Tr>
                    </Thead>
                    {reportSnapshots.length === 0 && (
                        <Tbody>
                            <Tr>
                                <Td colSpan={5}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No report jobs found"
                                            headingLevel="h2"
                                        >
                                            <Text>Clear any search value and try again</Text>
                                            <Button
                                                variant="link"
                                                onClick={() => {
                                                    setFilteredStatuses([]);
                                                    setPage(1);
                                                }}
                                            >
                                                Clear filters
                                            </Button>
                                        </EmptyStateTemplate>
                                    </Bullseye>
                                </Td>
                            </Tr>
                        </Tbody>
                    )}
                    {reportSnapshots.map((reportSnapshot, rowIndex) => {
                        const {
                            reportConfigId,
                            reportJobId,
                            name,
                            description,
                            vulnReportFilters,
                            collectionSnapshot,
                            schedule,
                            notifiers,
                            reportStatus,
                            user,
                            isDownloadAvailable,
                        } = reportSnapshot;
                        const isExpanded = expandedRowSet.has(reportJobId);
                        const reportConfiguration: ReportConfiguration = {
                            id: reportConfigId,
                            name,
                            description,
                            type: 'VULNERABILITY',
                            vulnReportFilters,
                            notifiers,
                            schedule,
                            resourceScope: {
                                collectionScope: {
                                    collectionId: collectionSnapshot.id,
                                    collectionName: collectionSnapshot.name,
                                },
                            },
                        };
                        const formValues =
                            getReportFormValuesFromConfiguration(reportConfiguration);
                        const areDownloadActionsDisabled = currentUser.userId !== user.id;

                        function onDownload() {
                            return saveFile({
                                method: 'get',
                                url: `/api/reports/jobs/download?id=${reportJobId}`,
                                data: null,
                                timeout: 300000,
                                name: `${name}.zip`,
                            });
                        }

                        const rowActions = [
                            {
                                title: (
                                    <span className="pf-u-danger-color-100">Delete download</span>
                                ),
                                onClick: (event) => {
                                    event.preventDefault();
                                    openDeleteDownloadModal(reportJobId);
                                },
                            },
                        ];

                        return (
                            <Tbody key={reportJobId} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(reportJobId),
                                        }}
                                    />
                                    <Td dataLabel="Completed">
                                        {reportStatus.completedAt
                                            ? getDateTime(reportStatus.completedAt)
                                            : '-'}
                                    </Td>
                                    <Td dataLabel="Status">
                                        <ReportJobStatus
                                            reportStatus={reportSnapshot.reportStatus}
                                            isDownloadAvailable={reportSnapshot.isDownloadAvailable}
                                            areDownloadActionsDisabled={areDownloadActionsDisabled}
                                            onDownload={onDownload}
                                        />
                                    </Td>
                                    <Td dataLabel="Requester">{user.name}</Td>
                                    <Td isActionCell>
                                        {isDownloadAvailable && (
                                            <ActionsColumn
                                                items={rowActions}
                                                isDisabled={areDownloadActionsDisabled}
                                            />
                                        )}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td colSpan={5}>
                                        <ExpandableRowContent>
                                            <Card className="pf-u-m-md pf-u-p-md" isFlat>
                                                <Flex>
                                                    <FlexItem>
                                                        <JobDetails
                                                            reportSnapshot={reportSnapshot}
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-u-my-md"
                                                    />
                                                    <FlexItem>
                                                        <ReportParametersDetails
                                                            formValues={formValues}
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-u-my-md"
                                                    />
                                                    <FlexItem>
                                                        <DeliveryDestinationsDetails
                                                            formValues={formValues}
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-u-my-md"
                                                    />
                                                    <FlexItem>
                                                        <ScheduleDetails formValues={formValues} />
                                                    </FlexItem>
                                                </Flex>
                                            </Card>
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </TableComposable>
            )}
            <DeleteModal
                title="Delete downloadable report?"
                isOpen={isDeleteDownloadModalOpen}
                onClose={closeDeleteDownloadModal}
                isDeleting={isDeletingDownload}
                onDelete={onDeleteDownload}
            >
                <AlertGroup>
                    {deleteDownloadError && (
                        <Alert
                            isInline
                            variant={AlertVariant.danger}
                            title={deleteDownloadError}
                            className="pf-u-mb-sm"
                        />
                    )}
                </AlertGroup>
                <p>
                    All data in this downloadable report will be deleted. Regenerating a
                    downloadable report will require the download process to start over.
                </p>
            </DeleteModal>
        </>
    );
}

export default ReportJobs;

import { useState } from 'react';
import {
    ActionsColumn,
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import {
    Alert,
    AlertGroup,
    Bullseye,
    Button,
    Card,
    Content,
    Divider,
    Flex,
    FlexItem,
    Pagination,
    SelectOption,
    Spinner,
    Switch,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, FilterIcon } from '@patternfly/react-icons';

import type { ConfiguredReportSnapshot, ReportConfiguration } from 'services/ReportsService.types';
import { getDateTime } from 'utils/dateUtils';
import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import useInterval from 'hooks/useInterval';
import useURLSort from 'hooks/useURLSort';
import { deleteDownloadableReport, downloadReportByJobId } from 'services/ReportsService';
import useAuthStatus from 'hooks/useAuthStatus';

import DeleteModal from 'Components/PatternFly/DeleteModal';
import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import type { TemplatePreviewArgs } from 'Components/EmailTemplate/EmailTemplateModal';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';

import { runStates } from 'types/reportJob';
import type { RunState } from 'types/reportJob';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';

import { getRequestQueryString } from '../api/apiUtils';
import useFetchReportHistory from '../api/useFetchReportHistory';
import EmailTemplatePreview from '../components/EmailTemplatePreview';
import ReportParametersDetails from '../components/ReportParametersDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import { defaultEmailBody, getDefaultEmailSubject } from '../forms/emailTemplateFormUtils';
import useDeleteDownloadModal from '../hooks/useDeleteDownloadModal';
import { getReportFormValuesFromConfiguration } from '../utils';
import JobDetails from './JobDetails';

export type ReportJobsProps = {
    reportId: string;
};

const sortOptions = {
    sortFields: ['Report Completion Time'],
    defaultSortOption: { field: 'Report Completion Time', direction: 'desc' } as const,
};

const headingLevel = 'h2';

const onDownload = (snapshot: ConfiguredReportSnapshot) => () => {
    const { reportJobId, name, reportStatus } = snapshot;
    const { completedAt } = reportStatus;
    const filename = `${name}-${completedAt}`;
    return downloadReportByJobId({
        reportJobId,
        filename,
        fileExtension: 'zip',
    });
};

function ReportJobs({ reportId }: ReportJobsProps) {
    const { currentUser } = useAuthStatus();
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const [filteredStatuses, setFilteredStatuses] = useState<RunState[]>([]);
    const [showOnlyMyJobs, setShowOnlyMyJobs] = useState<boolean>(false);
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
        deleteDownloadFunc: deleteDownloadableReport,
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
                            ariaLabel="Report status filter"
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
                                Report download is ready
                            </SelectOption>
                            <SelectOption value={runStates.DELIVERED}>
                                Report successfully sent
                            </SelectOption>
                            <SelectOption value={runStates.FAILURE}>
                                Report failed to generate
                            </SelectOption>
                        </CheckboxSelect>
                    </ToolbarItem>
                    <ToolbarItem className="pf-v6-u-flex-grow-1">
                        <Switch
                            id="view-only-my-jobs"
                            label="View only my jobs"
                            isChecked={showOnlyMyJobs}
                            onChange={(_event, checked: boolean) => handleChange(checked)}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignEnd' }}>
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
                <Bullseye className="pf-v6-u-background-color-100">
                    <EmptyStateTemplate
                        title="Error loading report jobs"
                        headingLevel="h2"
                        icon={ExclamationCircleIcon}
                        iconClassName="pf-v6-u-danger-color-100"
                    >
                        {error}
                    </EmptyStateTemplate>
                </Bullseye>
            )}
            {isLoading && !reportSnapshots && (
                <Bullseye className="pf-v6-u-background-color-100 pf-v6-u-p-lg">
                    <Spinner aria-label="Loading report jobs" />
                </Bullseye>
            )}
            {reportSnapshots && (
                <Table aria-label="Simple table" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>
                                <span className="pf-v5-screen-reader">Row expansion</span>
                            </Th>
                            <Th width={25} sort={getSortParams('Report Completion Time')}>
                                Completed
                            </Th>
                            <Th width={25}>Status</Th>
                            <Th width={50}>Requester</Th>
                            <Th>
                                <span className="pf-v5-screen-reader">Row actions</span>
                            </Th>
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
                                            <Content component="p">
                                                Clear any search value and try again
                                            </Content>
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
                            description: description ?? '',
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

                        const rowActions = [
                            {
                                title: (
                                    <span className="pf-v6-u-danger-color-100">
                                        Delete download
                                    </span>
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
                                            onDownload={onDownload(reportSnapshot)}
                                        />
                                    </Td>
                                    <Td dataLabel="Requester">{user.name}</Td>
                                    <Td isActionCell>
                                        {isDownloadAvailable && (
                                            <ActionsColumn
                                                // menuAppendTo={() => document.body}
                                                items={rowActions}
                                                isDisabled={areDownloadActionsDisabled}
                                            />
                                        )}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td colSpan={5}>
                                        <ExpandableRowContent>
                                            <Card className="pf-v6-u-m-md pf-v6-u-p-md">
                                                <Flex>
                                                    <FlexItem>
                                                        <JobDetails
                                                            reportStatus={reportStatus}
                                                            isDownloadAvailable={
                                                                isDownloadAvailable
                                                            }
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-v6-u-my-md"
                                                    />
                                                    <FlexItem>
                                                        <ReportParametersDetails
                                                            headingLevel={headingLevel}
                                                            formValues={formValues}
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-v6-u-my-md"
                                                    />
                                                    <FlexItem>
                                                        <NotifierConfigurationView
                                                            headingLevel={headingLevel}
                                                            customBodyDefault={defaultEmailBody}
                                                            customSubjectDefault={getDefaultEmailSubject(
                                                                formValues.reportParameters
                                                                    .reportName,
                                                                formValues.reportParameters
                                                                    .reportScope?.name
                                                            )}
                                                            notifierConfigurations={
                                                                formValues.deliveryDestinations
                                                            }
                                                            renderTemplatePreview={({
                                                                customBody,
                                                                customSubject,
                                                                customSubjectDefault,
                                                            }: TemplatePreviewArgs) => (
                                                                <EmailTemplatePreview
                                                                    emailSubject={customSubject}
                                                                    emailBody={customBody}
                                                                    defaultEmailSubject={
                                                                        customSubjectDefault
                                                                    }
                                                                    reportParameters={
                                                                        formValues.reportParameters
                                                                    }
                                                                />
                                                            )}
                                                        />
                                                    </FlexItem>
                                                    <Divider
                                                        component="div"
                                                        className="pf-v6-u-my-md"
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
                </Table>
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
                            variant="danger"
                            title={deleteDownloadError}
                            component="p"
                            className="pf-v6-u-mb-sm"
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

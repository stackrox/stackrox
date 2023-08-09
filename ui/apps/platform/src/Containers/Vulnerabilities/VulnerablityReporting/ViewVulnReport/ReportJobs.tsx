import React, { useState } from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import {
    Bullseye,
    Card,
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
import { CubesIcon, ExclamationCircleIcon, FilterIcon } from '@patternfly/react-icons';

import { ReportConfiguration, RunState, runStates } from 'services/ReportsService.types';
import { getDateTime } from 'utils/dateUtils';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import useFetchReportHistory from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReportHistory';
import { getRequestQueryString } from 'Containers/Vulnerabilities/VulnerablityReporting/api/apiUtils';

import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';
import ReportJobStatus from './ReportJobStatus';
import JobDetails from './JobDetails';

export type RunHistoryProps = {
    reportId: string;
};

function ReportJobs({ reportId }: RunHistoryProps) {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const [filteredStatuses, setFilteredStatuses] = useState<RunState[]>([]);
    const [showOnlyMyJobs, setShowOnlyMyJobs] = React.useState<boolean>(false);
    const expandedRowSet = useSet<string>();

    const query = getRequestQueryString({
        'Run state': filteredStatuses,
    });

    const { reportSnapshots, isLoading, error } = useFetchReportHistory({
        id: reportId,
        query,
        page,
        perPage,
        showMyHistory: showOnlyMyJobs,
    });

    const handleChange = (checked: boolean) => {
        setShowOnlyMyJobs(checked);
    };

    if (error) {
        return (
            <NotFoundMessage
                title="Error fetching report history"
                message={error || 'No data available'}
            />
        );
    }

    if (!isLoading && reportSnapshots.length === 0) {
        return (
            <Bullseye>
                <EmptyStateTemplate title="No run history" headingLevel="h2" icon={CubesIcon} />
            </Bullseye>
        );
    }

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
                            }}
                            placeholderText="Filter by status"
                        >
                            <SelectOption value={runStates.PREPARING}>Preparing</SelectOption>
                            <SelectOption value={runStates.WAITING}>Waiting</SelectOption>
                            <SelectOption value={runStates.SUCCESS}>Successful</SelectOption>
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
            {isLoading && (
                <Bullseye className="pf-u-background-color-100 pf-u-p-lg">
                    <Spinner aria-label="Loading report jobs" />
                </Bullseye>
            )}
            {!error && !isLoading && (
                <TableComposable aria-label="Simple table" variant="compact">
                    <Thead>
                        <Tr>
                            <Td>{/* Header for expanded column */}</Td>
                            <Th>Completed</Th>
                            <Th>Status</Th>
                            <Th>Requestor</Th>
                        </Tr>
                    </Thead>
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
                                        <ReportJobStatus reportSnapshot={reportSnapshot} />
                                    </Td>
                                    <Td dataLabel="Requester">{user.name}</Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td colSpan={4}>
                                        <Card className="pf-u-m-md pf-u-p-md" isFlat>
                                            <Flex direction={{ default: 'row' }}>
                                                <FlexItem flex={{ default: 'flex_1' }}>
                                                    <JobDetails reportStatus={reportStatus} />
                                                </FlexItem>
                                                <Divider
                                                    orientation={{
                                                        default: 'vertical',
                                                    }}
                                                />
                                                <FlexItem flex={{ default: 'flex_2' }}>
                                                    <ReportParametersDetails
                                                        formValues={formValues}
                                                    />
                                                    <Divider
                                                        component="div"
                                                        className="pf-u-py-md"
                                                    />
                                                    <DeliveryDestinationsDetails
                                                        formValues={formValues}
                                                    />
                                                    <Divider
                                                        component="div"
                                                        className="pf-u-py-md"
                                                    />
                                                    <ScheduleDetails formValues={formValues} />
                                                </FlexItem>
                                            </Flex>
                                        </Card>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </TableComposable>
            )}
        </>
    );
}

export default ReportJobs;

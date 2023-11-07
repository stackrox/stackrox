/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useEffect, useCallback } from 'react';
import { generatePath, Link } from 'react-router-dom';
import { format } from 'date-fns';
import {
    Alert,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Pagination,
    Spinner,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import {
    ActionsColumn,
    IAction,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import { complianceEnhancedScanConfigsBasePath } from 'routePaths';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getScanConfigs, Schedule } from 'services/ComplianceEnhancedService';
import { SortOption } from 'types/table';
import { displayOnlyItemOrItemCount } from 'utils/textUtils';
import { getDayOfMonthWithOrdinal, getTimeHoursMinutes } from 'utils/dateUtils';

import ScanConfigsHeader from '../ScanConfigsHeader';

type ScanConfigsTablePageProps = {
    hasWriteAccessForCompliance: boolean;
};

const CreateReportsButton = () => {
    return (
        <Link to={`${complianceEnhancedScanConfigsBasePath}/?action=create`}>
            <Button variant="primary">Create scan schedule</Button>
        </Link>
    );
};

const sortFields = ['Name', 'Last Run'];
const defaultSortOption = { field: 'Name', direction: 'asc' } as SortOption;

function ScanConfigsTablePage({
    hasWriteAccessForCompliance,
}: ScanConfigsTablePageProps): React.ReactElement {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });

    const listQuery = useCallback(
        () => getScanConfigs(sortOption, page - 1, perPage),
        [sortOption, page, perPage]
    );
    const { data: scanSchedules, loading: isLoading, error } = useRestQuery(listQuery);

    const formatScanSchedule = (schedule: Schedule) => {
        const daysOfWeekMap = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

        const formatDays = (days: string[]): string => {
            if (days.length === 1) {
                return days[0];
            }
            if (days.length === 2) {
                return days.join(' and ');
            }
            return `${days.slice(0, -1).join(', ')}, and ${days[days.length - 1]}`;
        };

        // arbitrary date, we only care about the time
        const date = new Date(2000, 0, 0, schedule.hour, schedule.minute);
        const timeString = getTimeHoursMinutes(date);

        switch (schedule.intervalType) {
            case 'DAILY':
                return `Daily at ${timeString}`;
            case 'WEEKLY': {
                const daysOfWeek = schedule.daysOfWeek.days.map((day) => daysOfWeekMap[day]);
                return `Every ${formatDays(daysOfWeek)} at ${timeString}`;
            }
            case 'MONTHLY': {
                const formattedDaysOfMonth =
                    schedule.daysOfMonth.days.map(getDayOfMonthWithOrdinal);
                return `Monthly on the ${formatDays(formattedDaysOfMonth)} at ${timeString}`;
            }
            default:
                return 'Invalid Schedule';
        }
    };

    const scanConfigActions = (): IAction[] => [
        {
            title: 'Edit schedule',
        },
        {
            title: 'Delete schedule',
        },
    ];

    const renderTableContent = () => {
        return scanSchedules?.map(
            ({ id, scanName, scanConfig, lastUpdatedTime, clusterStatus }) => (
                <Tr key={id}>
                    <Td>
                        <Link
                            to={generatePath(complianceEnhancedScanConfigsBasePath, {
                                policyId: id,
                            })}
                        >
                            {scanName}
                        </Link>
                    </Td>
                    <Td>{formatScanSchedule(scanConfig.scanSchedule)}</Td>
                    <Td>{format(lastUpdatedTime, 'DD MMM YYYY, h:mm:ss A')}</Td>
                    <Td>
                        {displayOnlyItemOrItemCount(
                            clusterStatus.map((cluster) => cluster.clusterName),
                            'clusters'
                        )}
                    </Td>
                    <Td>{displayOnlyItemOrItemCount(scanConfig.profiles, 'profiles')}</Td>
                    <Td isActionCell>
                        <ActionsColumn items={scanConfigActions()} />
                    </Td>
                </Tr>
            )
        );
    };

    const renderLoadingContent = () => (
        <Tr>
            <Td colSpan={8}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={6}>
                <Bullseye>
                    <EmptyStateTemplate
                        title="No scan schedules"
                        headingLevel="h2"
                        icon={OutlinedClockIcon}
                    >
                        {hasWriteAccessForCompliance && (
                            <Flex direction={{ default: 'column' }}>
                                <FlexItem>
                                    <Text>Create one to get started</Text>
                                </FlexItem>
                                <FlexItem>
                                    <CreateReportsButton />
                                </FlexItem>
                            </Flex>
                        )}
                    </EmptyStateTemplate>
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderTableBodyContent = () => {
        if (isLoading) {
            return renderLoadingContent();
        }
        if (scanSchedules && scanSchedules.length > 0) {
            return renderTableContent();
        }
        if (scanSchedules && scanSchedules.length === 0) {
            return renderEmptyContent();
        }
        return null;
    };

    return (
        <>
            <ScanConfigsHeader
                actions={hasWriteAccessForCompliance ? <CreateReportsButton /> : <></>}
                description="Configure scan schedules bound to clusters and policies."
            />
            <Divider component="div" />
            {error ? (
                <PageSection variant="light" isFilled id="policies-table-error">
                    <Bullseye>
                        <Alert variant="danger" title={error} />
                    </Bullseye>
                </PageSection>
            ) : (
                <PageSection>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                                <Pagination
                                    isCompact
                                    itemCount={scanSchedules ? scanSchedules.length : 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>

                    <TableComposable>
                        <Thead noWrap>
                            <Tr>
                                <Th sort={getSortParams('Name')}>Name</Th>
                                <Th>Schedule</Th>
                                <Th sort={getSortParams('Last Run')}>Last Run</Th>
                                <Th>Clusters</Th>
                                <Th>Profiles</Th>
                                <Td />
                            </Tr>
                        </Thead>
                        <Tbody>{renderTableBodyContent()}</Tbody>
                    </TableComposable>
                </PageSection>
            )}
        </>
    );
}

export default ScanConfigsTablePage;

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

import {
    complianceEnhancedScanConfigsPath,
    complianceEnhancedScanConfigDetailPath,
    complianceEnhancedCoveragePath,
} from 'routePaths';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import TabNavHeader from 'Components/TabNav/TabNavHeader';
import TabNavSubHeader from 'Components/TabNav/TabNavSubHeader';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getScanConfigs, Schedule } from 'services/ComplianceEnhancedService';
import { SortOption } from 'types/table';
import { displayOnlyItemOrItemCount } from 'utils/textUtils';
import { getDayOfMonthWithOrdinal, getTimeHoursMinutes } from 'utils/dateUtils';

import { formatScanSchedule } from '../compliance.scanConfigs.utils';

type ScanConfigsTablePageProps = {
    hasWriteAccessForCompliance: boolean;
};

const CreateScanConfigButton = () => {
    return (
        <Link to={`${complianceEnhancedScanConfigsPath}?action=create`}>
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
                            to={generatePath(complianceEnhancedScanConfigDetailPath, {
                                scanConfigId: id,
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
            <Td colSpan={5}>
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            </Td>
        </Tr>
    );

    const renderEmptyContent = () => (
        <Tr>
            <Td colSpan={5}>
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
                                    <CreateScanConfigButton />
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
        return renderEmptyContent();
    };

    return (
        <>
            <TabNavHeader
                currentTabTitle="Schedules"
                tabLinks={[
                    { title: 'Coverage', href: complianceEnhancedCoveragePath },
                    { title: 'Schedules', href: complianceEnhancedScanConfigsPath },
                ]}
                pageTitle="Compliance - Cluster compliance"
                mainTitle="Cluster compliance"
            />
            <Divider component="div" />
            <TabNavSubHeader
                actions={hasWriteAccessForCompliance ? <CreateScanConfigButton /> : <></>}
                description="Configure a scan schedule to run profile compliance checks on selected clusters"
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
                                <Th sort={getSortParams('Last Run')}>Last run</Th>
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
